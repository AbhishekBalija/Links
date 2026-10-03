package auth

// What Home shows those who bring others in: who was added by a class
// list or a staff invite and has not signed in yet, and the latest imports.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

const (
	// waitingOnHome caps the list; waiting_count always has the full number.
	waitingOnHome = 50
	importsOnHome = 5
)

// WaitingPerson is someone a class list or a staff invite let in who has not
// signed in yet.
type WaitingPerson struct {
	UserID         string    `json:"user_id" gorm:"column:user_id"`
	FullName       string    `json:"full_name" gorm:"column:full_name"`
	Email          string    `json:"email" gorm:"column:email"`
	Kind           string    `json:"kind" gorm:"column:kind"`
	Role           string    `json:"role,omitempty" gorm:"column:role"`
	USN            string    `json:"usn,omitempty" gorm:"column:usn"`
	DepartmentCode string    `json:"department_code" gorm:"column:department_code"`
	BatchYear      int       `json:"batch_year,omitempty" gorm:"column:batch_year"`
	AddedAt        time.Time `json:"added_at" gorm:"column:added_at"`
	AddedBy        AddedBy   `json:"added_by" gorm:"-"`
	AddedByName    string    `json:"-" gorm:"column:added_by_name"`
}

// AddedBy is who imported or invited someone.
type AddedBy struct {
	FullName string `json:"full_name"`
}

// NotSignedInFilter narrows the people waiting for a first sign-in: the
// actor's scope, students or staff, and the page after a cursor.
type NotSignedInFilter struct {
	Anywhere      bool
	DepartmentIDs []string
	Kind          string
	After         *NotSignedInCursor
	Limit         int
}

type NotSignedInCursor struct {
	At time.Time
	ID string
}

// WaitingList is the people waiting for a first sign-in, oldest first.
type WaitingList struct {
	Total  int
	People []WaitingPerson
}

// ImportBatch is how many students one import created in a Department and
// Batch.
type ImportBatch struct {
	DepartmentCode string `json:"department_code"`
	BatchYear      int    `json:"batch_year"`
	Created        int    `json:"created"`
}

// ImportedBy is who ran an import.
type ImportedBy struct {
	FullName string `json:"full_name"`
}

// ImportRun is one class-list import as the audit log recorded it. Rows and
// Failed count the whole file; Created and Batches only the viewer's scope.
type ImportRun struct {
	ImportedAt time.Time     `json:"imported_at"`
	ImportedBy ImportedBy    `json:"imported_by"`
	Rows       int           `json:"rows"`
	Created    int           `json:"created"`
	Failed     int           `json:"failed"`
	Batches    []ImportBatch `json:"batches"`
}

// ListsSummary is the HOD's, principal's and admins' view of who has
// been let in but not arrived, and of the latest imports.
type ListsSummary struct {
	WaitingCount  int             `json:"waiting_count"`
	Waiting       []WaitingPerson `json:"waiting"`
	HasMore       bool            `json:"has_more"`
	RecentImports []ImportRun     `json:"recent_imports"`
}

// importAudit is the metadata of a students_imported audit row.
type importAudit struct {
	Rows    int           `json:"rows"`
	Created int           `json:"created"`
	Failed  int           `json:"failed"`
	Batches []ImportBatch `json:"batches"`
}

// ImportAuditRow is a students_imported audit row with the importer's name.
type ImportAuditRow struct {
	At       time.Time `gorm:"column:at"`
	FullName string    `gorm:"column:full_name"`
	Metadata string    `gorm:"column:metadata"`
}

// ListsSummary reads who is waiting for a first sign-in and the latest
// imports: everywhere for the principal and admins, the HOD's own
// Departments otherwise. It is nil for anyone who doesn't bring people in.
func (s *authService) ListsSummary(ctx context.Context, actorID string) (*ListsSummary, error) {
	anywhere, scope, err := s.departmentScope(ctx, actorID, accessRefusal)
	var refused *apperrors.AppError
	if errors.As(err, &refused) && refused.HTTPStatus == http.StatusForbidden {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var departmentIDs []string
	if !anywhere {
		for id := range scope {
			departmentIDs = append(departmentIDs, id)
		}
		sort.Strings(departmentIDs)
	}

	waiting, err := s.userRepo.WaitingForFirstSignIn(ctx, anywhere, departmentIDs, waitingOnHome)
	if err != nil {
		return nil, fmt.Errorf("find people waiting for a first sign-in: %w", err)
	}
	audits, codes, err := s.userRepo.RecentImportAudits(ctx, anywhere, departmentIDs, importsOnHome)
	if err != nil {
		return nil, fmt.Errorf("find recent imports: %w", err)
	}
	summary := &ListsSummary{
		WaitingCount:  waiting.Total,
		Waiting:       waiting.People,
		HasMore:       waiting.Total > len(waiting.People),
		RecentImports: make([]ImportRun, 0, len(audits)),
	}
	if summary.Waiting == nil {
		summary.Waiting = []WaitingPerson{}
	}
	for _, audit := range audits {
		summary.RecentImports = append(summary.RecentImports, importRun(audit, anywhere, codes))
	}
	return summary, nil
}

// importRun reads one audit row, narrowing its batches and created count to
// the viewer's Departments. An audit row that can't be read is shown with
// the counts it has rather than hiding the import.
func importRun(audit ImportAuditRow, anywhere bool, codes map[string]bool) ImportRun {
	var meta importAudit
	_ = json.Unmarshal([]byte(audit.Metadata), &meta)
	run := ImportRun{
		ImportedAt: audit.At,
		ImportedBy: ImportedBy{FullName: audit.FullName},
		Rows:       meta.Rows,
		Created:    meta.Created,
		Failed:     meta.Failed,
		Batches:    []ImportBatch{},
	}
	if !anywhere {
		run.Created = 0
	}
	for _, batch := range meta.Batches {
		if !anywhere && !codes[batch.DepartmentCode] {
			continue
		}
		run.Batches = append(run.Batches, batch)
		if !anywhere {
			run.Created += batch.Created
		}
	}
	return run
}
