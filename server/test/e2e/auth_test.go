//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/app"
	"github.com/AbhishekBalija/Links/server/migrations"
	"github.com/AbhishekBalija/Links/server/pkg/config"
	"github.com/AbhishekBalija/Links/server/pkg/db"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

type responseEnvelope struct {
	Data json.RawMessage `json:"data"`
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type testServer struct {
	baseURL  string
	database *db.Database
	gormDB   *gorm.DB
	cleanup  func()
}

func findServerRoot() string {
	wd, _ := os.Getwd()
	for dir := wd; dir != "/"; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
	}
	return wd
}

func startServer(t *testing.T) *testServer {
	t.Helper()

	godotenv.Load(filepath.Join(findServerRoot(), ".env.local"))

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Port = "0"

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := database.Migrate(ctx, migrations.FS); err != nil {
		database.Close()
		t.Fatalf("run migrations: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler, err := app.NewServer(cfg, database, logger)
	if err != nil {
		database.Close()
		t.Fatalf("create server: %v", err)
	}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	portChan := make(chan string, 1)
	go func() {
		listener, err := net.Listen("tcp", server.Addr)
		if err != nil {
			t.Errorf("listen: %v", err)
			return
		}
		portChan <- strings.TrimPrefix(listener.Addr().String(), "[::]:")
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			t.Errorf("serve: %v", err)
		}
	}()

	return &testServer{
		baseURL:  fmt.Sprintf("http://localhost:%s", <-portChan),
		database: database,
		gormDB:   database.GORM(),
		cleanup: func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			server.Shutdown(ctx)
			database.Close()
		},
	}
}

func TestE2E_AuthFlow(t *testing.T) {
	ts := startServer(t)
	defer ts.cleanup()

	email := fmt.Sprintf("e2e-%s@test.com", uuid.New().String())

	// Step 1: Request access
	t.Run("request access creates pending user", func(t *testing.T) {
		body := map[string]interface{}{
			"email":     email,
			"password":  "TestPass123",
			"full_name": "E2E Test User",
		}
		status, resp := doPost(t, ts.baseURL+"/api/v1/auth/request-access", body)
		if status != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", status, string(resp))
		}
		var env responseEnvelope
		if err := json.Unmarshal(resp, &env); err != nil {
			t.Fatalf("parse response: %v", err)
		}
		var data map[string]interface{}
		if err := json.Unmarshal(env.Data, &data); err != nil {
			t.Fatalf("parse data: %v", err)
		}
		if data["user_id"] == "" {
			t.Fatal("expected user_id in response")
		}
		if data["status"] != "pending" {
			t.Fatalf("expected pending status, got %v", data["status"])
		}
	})

	// Step 2: Activate user directly and assign a role
	var userID string
	t.Run("activate user in DB", func(t *testing.T) {
		var u struct {
			ID string
		}
		if err := ts.gormDB.Raw("SELECT id FROM users WHERE email = ?", email).Scan(&u).Error; err != nil {
			t.Fatalf("find user: %v", err)
		}
		userID = u.ID

		if err := ts.gormDB.Exec("UPDATE users SET status = 'active' WHERE id = ?", userID).Error; err != nil {
			t.Fatalf("activate user: %v", err)
		}
		if err := ts.gormDB.Exec(
			`INSERT INTO role_assignments (id, user_id, role, scope_type, starts_at, created_at)
			 VALUES (?, ?, 'student', 'global', now(), now())`,
			uuid.New().String(), userID,
		).Error; err != nil {
			t.Fatalf("assign role: %v", err)
		}
	})

	// Step 3: Login
	var accessToken string
	t.Run("login returns tokens", func(t *testing.T) {
		body := map[string]interface{}{
			"email":    email,
			"password": "TestPass123",
		}
		status, resp := doPost(t, ts.baseURL+"/api/v1/auth/login", body)
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", status, string(resp))
		}
		var env responseEnvelope
		if err := json.Unmarshal(resp, &env); err != nil {
			t.Fatalf("parse response: %v", err)
		}
		var data struct {
			AccessToken string `json:"access_token"`
			ExpiresIn   int    `json:"expires_in"`
		}
		if err := json.Unmarshal(env.Data, &data); err != nil {
			t.Fatalf("parse data: %v", err)
		}
		if data.AccessToken == "" {
			t.Fatal("expected access_token")
		}
		if data.ExpiresIn <= 0 {
			t.Fatal("expected positive expires_in")
		}
		accessToken = data.AccessToken
	})

	// Step 4: GET /me with valid token
	t.Run("me returns user info", func(t *testing.T) {
		req, _ := http.NewRequest("GET", ts.baseURL+"/api/v1/me", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", res.StatusCode, string(body))
		}
		var env responseEnvelope
		if err := json.Unmarshal(body, &env); err != nil {
			t.Fatalf("parse response: %v", err)
		}
		var data struct {
			UserID string   `json:"user_id"`
			Email  string   `json:"email"`
			Roles  []string `json:"roles"`
		}
		if err := json.Unmarshal(env.Data, &data); err != nil {
			t.Fatalf("parse data: %v", err)
		}
		if data.UserID != userID {
			t.Fatalf("expected user_id %s, got %s", userID, data.UserID)
		}
		if len(data.Roles) != 1 || data.Roles[0] != "student" {
			t.Fatalf("expected [student] roles, got %v", data.Roles)
		}
	})

	// Step 5: GET /me without token -> 401
	t.Run("me without token returns 401", func(t *testing.T) {
		res, err := http.Get(ts.baseURL + "/api/v1/me")
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()

		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", res.StatusCode, string(body))
		}
	})
}

func doPost(t *testing.T, url string, body interface{}) (int, []byte) {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	res, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer res.Body.Close()
	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return res.StatusCode, respBody
}
