// Package apitest runs the real API router against an isolated Postgres
// schema, so tests exercise handlers, services and SQL together through HTTP.
//
// Tests skip unless TEST_DATABASE_URL is set. Point it at a throwaway local or
// CI database, never at the Neon dev or production branches: each test creates
// its own schema there and drops it afterwards.
package apitest

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/app"
	"github.com/AbhishekBalija/Links/server/internal/auth"
	"github.com/AbhishekBalija/Links/server/migrations"
	"github.com/AbhishekBalija/Links/server/pkg/config"
	"github.com/AbhishekBalija/Links/server/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	accessSecret  = "apitest-access-secret"
	refreshSecret = "apitest-refresh-secret"
)

// Harness is one test's API server and database schema.
type Harness struct {
	router   *gin.Engine
	database *db.Database
	tokenCfg auth.TokenConfig
	nextRoll int
	// Outbox holds the emails the API sent.
	Outbox *Outbox
	google *googleSigner
}

// New creates a fresh schema, applies every migration to it, and builds the
// API router on top. The schema is dropped when the test finishes, pass or fail.
func New(t *testing.T) *Harness {
	t.Helper()
	return NewWith(t, nil)
}

// NewWith is New with a chance to change the server's config first, for
// tests about config-gated routes.
func NewWith(t *testing.T, configure func(*config.Config)) *Harness {
	t.Helper()
	baseURL := os.Getenv("TEST_DATABASE_URL")
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping API test against Postgres")
	}

	schema := "apitest_" + randomHex(t, 6)
	admin := openDatabase(t, baseURL)
	if err := admin.GORM().Exec(fmt.Sprintf(`CREATE SCHEMA %q`, schema)).Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		if err := admin.GORM().Exec(fmt.Sprintf(`DROP SCHEMA %q CASCADE`, schema)).Error; err != nil {
			t.Errorf("drop schema %s: %v", schema, err)
		}
		_ = admin.Close()
	})

	database := openDatabase(t, withSearchPath(t, baseURL, schema))
	t.Cleanup(func() { _ = database.Close() })

	// Refuse to go on if the connection isn't really in the test schema, so a
	// driver ignoring search_path can never write into a real schema.
	var current string
	if err := database.GORM().Raw(`SELECT current_schema()`).Scan(&current).Error; err != nil || current != schema {
		t.Fatalf("test connection uses schema %q, want %q: %v", current, schema, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := database.Migrate(ctx, migrations.FS); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	cfg := config.Config{
		GINMode:          gin.TestMode,
		RequestBodyLimit: 1 << 20,
		CORS:             config.CORSConfig{AllowedOrigins: []string{"http://localhost:5173"}},
		Auth: config.AuthConfig{
			JWTAccessSecret:  accessSecret,
			JWTRefreshSecret: refreshSecret,
			AccessTokenTTL:   15 * time.Minute,
			RefreshTokenTTL:  24 * time.Hour,
		},
		Google: config.GoogleConfig{ClientID: GoogleClientID},
	}
	if configure != nil {
		configure(&cfg)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	outbox := &Outbox{}
	codeSettings := auth.DefaultCodeSettings()
	// Tests don't wait out the reply floor that hides whether a code was sent.
	codeSettings.MinReplyTime = 0
	google := newGoogleSigner(t)
	router, err := app.NewServer(cfg, database, logger,
		app.WithMailer(outbox),
		app.WithCodeSettings(codeSettings),
		app.WithGoogleCertsClient(google.certsClient()),
	)
	if err != nil {
		t.Fatalf("build router: %v", err)
	}

	return &Harness{
		router:   router,
		database: database,
		Outbox:   outbox,
		google:   google,
		tokenCfg: auth.TokenConfig{
			AccessSecret:  accessSecret,
			RefreshSecret: refreshSecret,
			AccessTTL:     15 * time.Minute,
			RefreshTTL:    24 * time.Hour,
		},
	}
}

// DB gives seeding helpers direct access to the test schema. Tests should
// check behaviour through the API, not by querying tables.
func (h *Harness) DB() *gorm.DB {
	return h.database.GORM()
}

// RoleSeed is one role assignment. Leave DepartmentCode empty for a global role.
type RoleSeed struct {
	Role           string
	DepartmentCode string
}

// StudentSeed gives the user a Student identity in a seeded department.
type StudentSeed struct {
	DepartmentCode string
	BatchYear      int
}

// UserSeed describes an active, verified user to insert.
type UserSeed struct {
	Roles   []RoleSeed
	Student *StudentSeed
}

// User is a seeded user with an access token carrying their roles.
type User struct {
	ID    string
	Email string
	Token string
}

// SeedUser inserts an active user with the given roles and optional Student
// identity, and signs an access token for them.
func (h *Harness) SeedUser(t *testing.T, seed UserSeed) User {
	t.Helper()
	id := uuid.NewString()
	suffix := randomHex(t, 4)
	email := "user-" + suffix + "@apitest.local"
	h.exec(t, `INSERT INTO users (id, email, password_hash, status, is_verified, created_at, updated_at)
		VALUES (?, ?, 'not-a-real-hash', 'active', true, now(), now())`, id, email)
	h.exec(t, `INSERT INTO profiles (user_id, username, full_name) VALUES (?, ?, ?)`,
		id, "user_"+suffix, "Test User "+suffix)

	roleNames := make([]string, 0, len(seed.Roles))
	for _, role := range seed.Roles {
		if role.DepartmentCode == "" {
			h.exec(t, `INSERT INTO role_assignments (id, user_id, role, scope_type, starts_at, created_at)
				VALUES (?, ?, ?, 'global', now(), now())`, uuid.NewString(), id, role.Role)
		} else {
			h.exec(t, `INSERT INTO role_assignments (id, user_id, role, scope_type, scope_id, starts_at, created_at)
				VALUES (?, ?, ?, 'department', ?, now(), now())`,
				uuid.NewString(), id, role.Role, h.DepartmentID(t, role.DepartmentCode))
		}
		roleNames = append(roleNames, role.Role)
	}

	if seed.Student != nil {
		// A per-harness counter keeps USNs unique within the schema.
		h.nextRoll++
		usn := fmt.Sprintf("4MN%02d%s%03d", seed.Student.BatchYear%100, seed.Student.DepartmentCode, h.nextRoll)
		h.exec(t, `INSERT INTO student_identities (user_id, usn, department_id, batch_year) VALUES (?, ?, ?, ?)`,
			id, usn, h.DepartmentID(t, seed.Student.DepartmentCode), seed.Student.BatchYear)
	}

	token, err := auth.GenerateAccessToken(id, roleNames, h.tokenCfg)
	if err != nil {
		t.Fatalf("sign access token: %v", err)
	}
	return User{ID: id, Email: email, Token: token}
}

// SignIn gives a seeded user a real password, logs them in through the API and
// returns their refresh token cookie, for tests about sessions.
func (h *Harness) SignIn(t *testing.T, user User) *http.Cookie {
	t.Helper()
	const password = "Apitest-password-1"
	hash, err := auth.NewArgon2PasswordHasher().Hash(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	h.exec(t, `UPDATE users SET password_hash = ? WHERE id = ?`, hash, user.ID)
	response := h.Do(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"email": user.Email, "password": password})
	if response.Status != http.StatusOK {
		t.Fatalf("login status = %d, want %d: %s", response.Status, http.StatusOK, response.Body)
	}
	for _, cookie := range response.Cookies() {
		if cookie.Name == "refresh_token" {
			return cookie
		}
	}
	t.Fatal("login set no refresh_token cookie")
	return nil
}

// TokenFor signs a fresh access token for an existing user with the given role
// names, the way a refresh would after their roles changed.
func (h *Harness) TokenFor(t *testing.T, userID string, roles ...string) string {
	t.Helper()
	token, err := auth.GenerateAccessToken(userID, roles, h.tokenCfg)
	if err != nil {
		t.Fatalf("sign access token: %v", err)
	}
	return token
}

// DepartmentID looks up a department's ID by its code.
func (h *Harness) DepartmentID(t *testing.T, code string) string {
	t.Helper()
	var id string
	if err := h.DB().Raw(`SELECT id FROM departments WHERE code = ?`, code).Scan(&id).Error; err != nil || id == "" {
		t.Fatalf("department %s not found: %v", code, err)
	}
	return id
}

// Response is what an API call returned.
type Response struct {
	Status int
	Body   string
	Header http.Header
}

// Cookies returns the cookies the response set.
func (r Response) Cookies() []*http.Cookie {
	return (&http.Response{Header: r.Header}).Cookies()
}

// Decode parses the response body as JSON into target.
func (r Response) Decode(t *testing.T, target any) {
	t.Helper()
	if err := json.Unmarshal([]byte(r.Body), target); err != nil {
		t.Fatalf("decode response %q: %v", r.Body, err)
	}
}

// Do calls the API. token may be empty for anonymous calls; body is sent as JSON when not nil.
func (h *Harness) Do(t *testing.T, method, path, token string, body any) Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode request body: %v", err)
		}
		reader = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, path, reader)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	return h.Send(request)
}

// Send runs a request built by the test, for calls that need their own
// headers or cookies.
func (h *Harness) Send(request *http.Request) Response {
	recorder := httptest.NewRecorder()
	h.router.ServeHTTP(recorder, request)
	return Response{Status: recorder.Code, Body: recorder.Body.String(), Header: recorder.Header()}
}

func (h *Harness) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	if err := h.DB().Exec(query, args...).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func openDatabase(t *testing.T, databaseURL string) *db.Database {
	t.Helper()
	database, err := db.New(config.Config{
		DatabaseURL: databaseURL,
		DatabasePool: config.DatabasePoolConfig{
			MaxOpenConns:    5,
			MaxIdleConns:    2,
			ConnMaxLifetime: time.Minute,
			ConnMaxIdleTime: time.Minute,
		},
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	return database
}

// withSearchPath makes every connection in the pool use the test schema.
func withSearchPath(t *testing.T, databaseURL, schema string) string {
	t.Helper()
	parsed, err := url.Parse(databaseURL)
	if err != nil || !strings.HasPrefix(parsed.Scheme, "postgres") {
		t.Fatalf("TEST_DATABASE_URL must be a postgres:// URL")
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func randomHex(t *testing.T, bytesLen int) string {
	t.Helper()
	buffer := make([]byte, bytesLen)
	if _, err := rand.Read(buffer); err != nil {
		t.Fatalf("random: %v", err)
	}
	return hex.EncodeToString(buffer)
}
