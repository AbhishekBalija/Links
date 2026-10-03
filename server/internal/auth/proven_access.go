package auth

// Accounts let in without a password (spec #129): an Access request from
// someone who proved their email but is on no list, and a staff invite. Both
// end up waiting for a first sign-in; the request first waits for Access
// approval (ADR 0025).

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

// RequestAccessWithProof sends an Access request for the email a request
// token proves. The Department and Batch come from the USN, and the request
// joins that Department's review queue.
func (s *authService) RequestAccessWithProof(ctx context.Context, input ProvenAccessRequestInput) (*RequestAccessResponse, error) {
	email, err := parseAccessRequestToken(s.tokenCfg, input.RequestToken)
	if err != nil {
		return nil, apperrors.NewUnauthenticated("the proof of your email has expired; sign in again to send the request")
	}
	fullName := strings.TrimSpace(input.FullName)
	if fullName == "" {
		return nil, apperrors.NewValidation("full_name is empty", map[string]string{"full_name": "required"})
	}
	usn := strings.ToUpper(strings.TrimSpace(input.USN))
	code, err := ValidateUSNFormat(usn)
	if err != nil {
		return nil, apperrors.NewValidation("invalid USN: "+err.Error(), map[string]string{"usn": err.Error()})
	}
	batchYear, err := BatchYearFromUSN(usn)
	if err != nil {
		return nil, apperrors.NewValidation("invalid USN: "+err.Error(), map[string]string{"usn": err.Error()})
	}
	department, err := s.userRepo.FindDepartmentByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("find department: %w", err)
	}
	if department == nil {
		return nil, apperrors.NewValidation("no department has the code "+code, map[string]string{"usn": "unknown department code " + code})
	}

	now := time.Now()
	user := &User{Email: &email, Status: UserStatusPending, CreatedAt: now, UpdatedAt: now}
	err = s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		if err := requireNewEmail(ctx, repos.Users, email); err != nil {
			return err
		}
		taken, err := repos.Users.USNExists(ctx, usn)
		if err != nil {
			return fmt.Errorf("find USN: %w", err)
		}
		if taken {
			return apperrors.NewConflict("the USN is already registered")
		}
		exists, err := repos.Users.LockDepartmentForShare(ctx, department.ID)
		if err != nil {
			return fmt.Errorf("lock department: %w", err)
		}
		if !exists {
			return apperrors.NewValidation("no department has the code "+code, nil)
		}
		if err := s.createAccount(ctx, repos.Users, user, fullName, now); err != nil {
			return err
		}
		identity := &StudentIdentity{UserID: user.ID, USN: usn, DepartmentID: department.ID, BatchYear: batchYear, CreatedAt: now, UpdatedAt: now}
		if err := repos.Users.CreateStudentIdentity(ctx, identity); err != nil {
			return fmt.Errorf("create student identity: %w", err)
		}
		return repos.AuditLogs.Create(ctx, &AuditLog{
			ActorID:      &user.ID,
			Action:       "access_requested",
			ResourceType: "user",
			ResourceID:   &user.ID,
			Metadata:     map[string]string{"usn": usn, "department_code": department.Code, "proof": "email"},
			CreatedAt:    now,
		})
	})
	if err := uniqueViolation(err); err != nil {
		return nil, err
	}
	return &RequestAccessResponse{UserID: user.ID, Status: string(user.Status)}, nil
}

// InviteStaff adds a staff member by email and role. The account waits for
// its first sign-in; the role follows role management's rules, so only an
// admin can invite an admin.
func (s *authService) InviteStaff(ctx context.Context, actorID string, input InviteStaffInput) (*RequestAccessResponse, error) {
	email := strings.TrimSpace(input.Email)
	if address, err := mail.ParseAddress(email); err != nil || address.Address != email {
		return nil, apperrors.NewValidation("email is not a valid address", map[string]string{"email": "invalid"})
	}
	fullName := strings.TrimSpace(input.FullName)
	if fullName == "" {
		return nil, apperrors.NewValidation("full_name is empty", map[string]string{"full_name": "required"})
	}
	grant := GrantRoleInput{Role: input.Role, ScopeType: input.ScopeType, ScopeID: input.ScopeID, Note: input.Note}
	if err := validateGrant(Role(grant.Role), ScopeType(grant.ScopeType), grant.ScopeID); err != nil {
		return nil, err
	}

	now := time.Now()
	user := &User{Email: &email, Status: UserStatusPending, IsVerified: true, CreatedBy: &actorID, CreatedAt: now, UpdatedAt: now}
	err := s.unitOfWork.WithinTransaction(ctx, func(repos AuthRepositories) error {
		if err := requireNewEmail(ctx, repos.Users, email); err != nil {
			return err
		}
		if err := s.createAccount(ctx, repos.Users, user, fullName, now); err != nil {
			return err
		}
		if _, err := grantRoleIn(ctx, repos, actorID, user.ID, grant, now, now, requireInviter); err != nil {
			return err
		}
		return repos.AuditLogs.Create(ctx, &AuditLog{
			ActorID:      &actorID,
			Action:       "user_invited",
			ResourceType: "user",
			ResourceID:   &user.ID,
			Metadata:     map[string]string{"role": grant.Role},
			CreatedAt:    now,
		})
	})
	if err := uniqueViolation(err); err != nil {
		return nil, err
	}
	return &RequestAccessResponse{UserID: user.ID, Status: string(user.Status)}, nil
}

func requireNewEmail(ctx context.Context, users UserRepository, email string) error {
	existing, err := users.FindByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("find email: %w", err)
	}
	if existing != nil {
		return apperrors.NewConflict("the email is already registered")
	}
	return nil
}

// createAccount creates a passwordless user and their profile.
func (s *authService) createAccount(ctx context.Context, users UserRepository, user *User, fullName string, now time.Time) error {
	if err := users.Create(ctx, user); err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	profile := &Profile{UserID: user.ID, FullName: fullName, PublicProfileEnabled: true, CreatedAt: now, UpdatedAt: now}
	if err := s.createProfileWithRetry(ctx, users, profile); err != nil {
		return fmt.Errorf("create profile: %w", err)
	}
	return nil
}

// uniqueViolation turns a lost race for an email or USN into the conflict
// the earlier check would have given.
func uniqueViolation(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "idx_users_email":
			return apperrors.NewConflict("the email is already registered")
		case "idx_student_identities_usn":
			return apperrors.NewConflict("the USN is already registered")
		}
	}
	return err
}
