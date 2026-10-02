package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var errActivationTokenUnavailable = errors.New("activation token is unavailable")
var errRefreshTokenUnavailable = errors.New("refresh token is unavailable")

// GormAuthUnitOfWork creates transaction-scoped auth repositories.
type GormAuthUnitOfWork struct {
	db *gorm.DB
}

func NewGormAuthUnitOfWork(db *gorm.DB) *GormAuthUnitOfWork {
	return &GormAuthUnitOfWork{db: db}
}

func (u *GormAuthUnitOfWork) WithinTransaction(ctx context.Context, fn func(AuthRepositories) error) error {
	return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(AuthRepositories{
			Users:         NewGormUserRepository(tx),
			RefreshTokens: NewGormRefreshTokenRepository(tx),
			Activations:   NewGormActivationTokenRepository(tx),
			SignInCodes:   NewGormSignInCodeRepository(tx),
			AuditLogs:     NewGormAuditLogRepository(tx),
		})
	})
}

// GormUserRepository implements UserRepository using GORM.
type GormUserRepository struct {
	db *gorm.DB
}

// NewGormUserRepository creates a new GormUserRepository.
func NewGormUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) Create(ctx context.Context, user *User) error {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *GormUserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("lower(email) = lower(?)", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

// FindByEmailForUpdate is FindByEmail with the row locked until the
// transaction ends.
func (r *GormUserRepository) FindByEmailForUpdate(ctx context.Context, email string) (*User, error) {
	var users []User
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("Profile").
		Preload("StudentIdentity").
		Where("lower(email) = lower(?)", email).
		Limit(1).
		Find(&users).Error
	if err != nil || len(users) == 0 {
		return nil, err
	}
	return &users[0], nil
}

func (r *GormUserRepository) FindByGoogleSubjectForUpdate(ctx context.Context, subject string) (*User, error) {
	var users []User
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("Profile").
		Preload("StudentIdentity").
		Where("google_subject = ?", subject).
		Limit(1).
		Find(&users).Error
	if err != nil || len(users) == 0 {
		return nil, err
	}
	return &users[0], nil
}

// SetGoogleSubject links a Google account to a user who has none yet.
func (r *GormUserRepository) SetGoogleSubject(ctx context.Context, userID, subject string) error {
	result := r.db.WithContext(ctx).Model(&User{}).
		Where("id = ? AND google_subject IS NULL", userID).
		Updates(map[string]any{"google_subject": subject, "updated_at": time.Now()})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("user %s already has a Google account linked", userID)
	}
	return nil
}

// CompleteFirstSignIn makes an account waiting for its first sign-in active.
func (r *GormUserRepository) CompleteFirstSignIn(ctx context.Context, userID string, at time.Time) error {
	result := r.db.WithContext(ctx).Model(&User{}).
		Where("id = ? AND status = ? AND is_verified", userID, UserStatusPending).
		Updates(map[string]any{"status": UserStatusActive, "first_signed_in_at": at, "updated_at": at})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("user %s is not waiting for a first sign-in", userID)
	}
	return nil
}

// ReturnForReview puts an account back to undecided, with no Google account
// linked, so nobody can sign into it until it is approved again.
func (r *GormUserRepository) ReturnForReview(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).
		Updates(map[string]any{
			"status":             UserStatusPending,
			"is_verified":        false,
			"google_subject":     nil,
			"first_signed_in_at": nil,
			"updated_at":         time.Now(),
		}).Error
}

func (r *GormUserRepository) FindByID(ctx context.Context, id string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).
		Preload("Profile").
		Preload("StudentIdentity").
		First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *GormUserRepository) FindByIDForUpdate(ctx context.Context, id string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("Profile").
		Preload("StudentIdentity").
		First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *GormUserRepository) Update(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *GormUserRepository) UpdateStatus(ctx context.Context, id string, status UserStatus) error {
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Update("status", string(status)).Error
}

func (r *GormUserRepository) FindEmailByUserID(ctx context.Context, userID string) (*string, error) {
	var email *string
	err := r.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Select("email").Scan(&email).Error
	return email, err
}

func (r *GormUserRepository) FindPhoneByUserID(ctx context.Context, userID string) (*string, error) {
	var phone *string
	err := r.db.WithContext(ctx).Model(&User{}).Where("id = ?", userID).Select("phone").Scan(&phone).Error
	return phone, err
}

func (r *GormUserRepository) FindDepartmentByCode(ctx context.Context, code string) (*Department, error) {
	var dept Department
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&dept).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &dept, err
}

func (r *GormUserRepository) FindDepartmentByID(ctx context.Context, id string) (*Department, error) {
	var dept Department
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&dept).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &dept, err
}

// LockDepartmentForShare reports whether the department exists and holds a
// share lock on it until the transaction ends, so the department can't be
// deleted while a role scoped to it is being created.
func (r *GormUserRepository) LockDepartmentForShare(ctx context.Context, id string) (bool, error) {
	var ids []string
	err := r.db.WithContext(ctx).
		Raw("SELECT id FROM departments WHERE id = ? FOR SHARE", id).
		Scan(&ids).Error
	return len(ids) > 0, err
}

func (r *GormUserRepository) CreateProfile(ctx context.Context, profile *Profile) error {
	return r.db.WithContext(ctx).Create(profile).Error
}

func (r *GormUserRepository) CreateStudentIdentity(ctx context.Context, identity *StudentIdentity) error {
	return r.db.WithContext(ctx).Create(identity).Error
}

// USNExists matches the case-insensitive unique index on student_identities.
func (r *GormUserRepository) USNExists(ctx context.Context, usn string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&StudentIdentity{}).Where("lower(usn) = lower(?)", usn).Count(&count).Error
	return count > 0, err
}

func (r *GormUserRepository) GetRoleAssignments(ctx context.Context, userID string) ([]RoleAssignment, error) {
	var roles []RoleAssignment
	// Only roles in effect now count: started, and not yet ended.
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND starts_at <= now() AND (ends_at IS NULL OR ends_at > now())", userID).
		Find(&roles).Error
	return roles, err
}

func (r *GormUserRepository) FindPendingUsers(ctx context.Context) ([]User, error) {
	var users []User
	err := r.db.WithContext(ctx).
		Preload("Profile").
		Preload("StudentIdentity").
		// An approved student stays pending until they activate, but no
		// longer waits for approval.
		Where("status = ? AND is_verified = false", UserStatusPending).
		Order("created_at asc").
		Find(&users).Error
	return users, err
}

func (r *GormUserRepository) ListRoleAssignments(ctx context.Context, userID string) ([]RoleAssignmentView, error) {
	var views []RoleAssignmentView
	err := r.db.WithContext(ctx).Raw(`
		SELECT ra.*, d.code AS department_code, d.name AS department_name
		FROM role_assignments ra
		LEFT JOIN departments d ON ra.scope_type = 'department' AND d.id = ra.scope_id
		WHERE ra.user_id = ?
		ORDER BY ra.starts_at DESC, ra.created_at DESC, ra.id`, userID).
		Scan(&views).Error
	return views, err
}

func (r *GormUserRepository) FindRoleAssignmentForUpdate(ctx context.Context, userID, id string) (*RoleAssignment, error) {
	var assignments []RoleAssignment
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND user_id = ?", id, userID).
		Limit(1).
		Find(&assignments).Error
	if err != nil || len(assignments) == 0 {
		return nil, err
	}
	return &assignments[0], nil
}

func (r *GormUserRepository) HasOverlappingAssignment(ctx context.Context, filter OverlapFilter) (bool, error) {
	query := r.db.WithContext(ctx).Model(&RoleAssignment{}).
		Where("role = ? AND scope_type = ?", filter.Role, filter.ScopeType).
		// Two ranges overlap when each starts before the other ends.
		Where("ends_at IS NULL OR ends_at > ?", filter.StartsAt)
	if filter.EndsAt != nil {
		query = query.Where("starts_at < ?", *filter.EndsAt)
	}
	if filter.ScopeID == nil {
		query = query.Where("scope_id IS NULL")
	} else {
		query = query.Where("scope_id = ?", *filter.ScopeID)
	}
	if filter.UserID != "" {
		query = query.Where("user_id = ?", filter.UserID)
	}
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

// LockDepartmentForUpdate reports whether the department exists and holds a
// row lock on it, so two HOD grants for one department run one after another.
func (r *GormUserRepository) LockDepartmentForUpdate(ctx context.Context, id string) (bool, error) {
	var ids []string
	err := r.db.WithContext(ctx).
		Raw("SELECT id FROM departments WHERE id = ? FOR UPDATE", id).
		Scan(&ids).Error
	return len(ids) > 0, err
}

// LockAdminAssignmentsInEffect locks every admin assignment in effect now held
// by an active user, so two admins can't end each other's role at once and
// leave the college with none.
func (r *GormUserRepository) LockAdminAssignmentsInEffect(ctx context.Context) ([]RoleAssignment, error) {
	var assignments []RoleAssignment
	err := r.db.WithContext(ctx).Raw(`
		SELECT ra.* FROM role_assignments ra
		JOIN users u ON u.id = ra.user_id
		WHERE ra.role = ? AND u.status = ?
		  AND ra.starts_at <= now() AND (ra.ends_at IS NULL OR ra.ends_at > now())
		FOR UPDATE OF ra`, RoleAdmin, UserStatusActive).
		Scan(&assignments).Error
	return assignments, err
}

func (r *GormUserRepository) EndRoleAssignment(ctx context.Context, id string, endsAt time.Time) error {
	return r.db.WithContext(ctx).Model(&RoleAssignment{}).Where("id = ?", id).Update("ends_at", endsAt).Error
}

// ClearDepartmentHOD removes the user as the department's named HOD, if they are.
func (r *GormUserRepository) ClearDepartmentHOD(ctx context.Context, departmentID, userID string) error {
	return r.db.WithContext(ctx).
		Exec("UPDATE departments SET hod_user_id = NULL, updated_at = now() WHERE id = ? AND hod_user_id = ?", departmentID, userID).
		Error
}

func (r *GormUserRepository) CreateRoleAssignment(ctx context.Context, ra *RoleAssignment) error {
	if ra.ID == "" {
		ra.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(ra).Error
}

// GormRefreshTokenRepository implements RefreshTokenRepository using GORM.
type GormRefreshTokenRepository struct {
	db *gorm.DB
}

// NewGormRefreshTokenRepository creates a new GormRefreshTokenRepository.
func NewGormRefreshTokenRepository(db *gorm.DB) *GormRefreshTokenRepository {
	return &GormRefreshTokenRepository{db: db}
}

func (r *GormRefreshTokenRepository) Create(ctx context.Context, token *RefreshToken) error {
	if token.ID == "" {
		token.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *GormRefreshTokenRepository) FindByHash(ctx context.Context, hash string) (*RefreshToken, error) {
	var token RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &token, err
}

func (r *GormRefreshTokenRepository) RevokeByHash(ctx context.Context, hash string) error {
	return r.db.WithContext(ctx).Model(&RefreshToken{}).Where("token_hash = ?", hash).Update("revoked_at", time.Now()).Error
}

func (r *GormRefreshTokenRepository) RevokeIfActive(ctx context.Context, hash string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&RefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", hash, now).
		Update("revoked_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("%w: token already revoked, expired, or missing", errRefreshTokenUnavailable)
	}
	return nil
}

// RevokeAllByUserID revokes the user's refresh tokens that are still usable,
// leaving already revoked ones with their original time.
func (r *GormRefreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Model(&RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}

// GormActivationTokenRepository implements ActivationTokenRepository.
type GormActivationTokenRepository struct {
	db *gorm.DB
}

func NewGormActivationTokenRepository(db *gorm.DB) *GormActivationTokenRepository {
	return &GormActivationTokenRepository{db: db}
}

func (r *GormActivationTokenRepository) Create(ctx context.Context, token *AccountActivationToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *GormActivationTokenRepository) FindLatestByUserID(ctx context.Context, userID string) (*AccountActivationToken, error) {
	var token AccountActivationToken
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &token, err
}

func (r *GormActivationTokenRepository) RevokeAllUnusedByUserID(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Model(&AccountActivationToken{}).
		Where("user_id = ? AND used_at IS NULL", userID).
		UpdateColumn("used_at", time.Now()).
		Error
}

func (r *GormActivationTokenRepository) FindByHash(ctx context.Context, hash string) (*AccountActivationToken, error) {
	var token AccountActivationToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &token, err
}

func (r *GormActivationTokenRepository) MarkUsed(ctx context.Context, id string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&AccountActivationToken{}).
		Where("id = ? AND used_at IS NULL AND expires_at > ?", id, now).
		Update("used_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("%w: token already used, expired, or missing", errActivationTokenUnavailable)
	}
	return nil
}

// GormAuditLogRepository implements AuditLogRepository using GORM.
type GormAuditLogRepository struct {
	db *gorm.DB
}

func NewGormAuditLogRepository(db *gorm.DB) *GormAuditLogRepository {
	return &GormAuditLogRepository{db: db}
}

func (r *GormAuditLogRepository) Create(ctx context.Context, log *AuditLog) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	if log.Metadata != nil {
		b, err := json.Marshal(log.Metadata)
		if err != nil {
			return err
		}
		s := string(b)
		log.MetadataJSON = &s
	}
	return r.db.WithContext(ctx).Create(log).Error
}

// GormSignInCodeRepository implements SignInCodeRepository.
type GormSignInCodeRepository struct {
	db *gorm.DB
}

func NewGormSignInCodeRepository(db *gorm.DB) *GormSignInCodeRepository {
	return &GormSignInCodeRepository{db: db}
}

func (r *GormSignInCodeRepository) LockEmail(ctx context.Context, emailHash string) error {
	return r.db.WithContext(ctx).Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?, 0))`, "sign-in-code:"+emailHash).Error
}

func (r *GormSignInCodeRepository) DeleteCreatedBefore(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).Where("created_at < ?", before).Delete(&SignInCode{}).Error
}

func (r *GormSignInCodeRepository) CountByEmailSince(ctx context.Context, emailHash string, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&SignInCode{}).Where("email_hash = ? AND created_at > ?", emailHash, since).Count(&count).Error
	return count, err
}

func (r *GormSignInCodeRepository) CountByIPSince(ctx context.Context, ipHash string, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&SignInCode{}).Where("ip_hash = ? AND created_at > ?", ipHash, since).Count(&count).Error
	return count, err
}

func (r *GormSignInCodeRepository) SumWrongTriesByEmailSince(ctx context.Context, emailHash string, since time.Time) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&SignInCode{}).Select("COALESCE(SUM(attempts), 0)").
		Where("email_hash = ? AND created_at > ?", emailHash, since).Scan(&total).Error
	return total, err
}

func (r *GormSignInCodeRepository) Create(ctx context.Context, code *SignInCode) error {
	return r.db.WithContext(ctx).Create(code).Error
}

func (r *GormSignInCodeRepository) FindForUpdate(ctx context.Context, id string) (*SignInCode, error) {
	var codes []SignInCode
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		Limit(1).
		Find(&codes).Error
	if err != nil || len(codes) == 0 {
		return nil, err
	}
	return &codes[0], nil
}

func (r *GormSignInCodeRepository) RecordWrongTry(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&SignInCode{}).Where("id = ?", id).
		UpdateColumn("attempts", gorm.Expr("attempts + 1")).Error
}

func (r *GormSignInCodeRepository) MarkUsed(ctx context.Context, id string, at time.Time) error {
	return r.db.WithContext(ctx).Model(&SignInCode{}).Where("id = ? AND used_at IS NULL", id).
		UpdateColumn("used_at", at).Error
}
