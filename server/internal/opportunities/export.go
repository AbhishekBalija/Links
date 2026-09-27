package opportunities

// Applicant export: placement staff download an Opportunity's applicants as
// CSV.

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/shared/csvsafe"
	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

func exportHeader() []string {
	return []string{"full_name", "email", "usn", "department", "batch_year", "mode", "status", "applied_at", "status_changed_at"}
}

// Export returns an Opportunity's applicants as CSV to placement staff,
// optionally one status only, and records the export in the audit log in
// the same transaction as reading the rows. It never includes phone numbers.
func (s *Service) Export(ctx context.Context, actorID, opportunityID, status string) ([]byte, error) {
	if err := s.requireStaff(ctx, actorID); err != nil {
		return nil, err
	}
	filter, err := s.applicantFilter(ctx, ApplicantQuery{Status: status})
	if err != nil {
		return nil, err
	}
	opportunity, err := s.repository.Find(ctx, opportunityID)
	if err != nil {
		return nil, fmt.Errorf("find opportunity: %w", err)
	}
	if opportunity == nil {
		return nil, apperrors.NewNotFound("opportunity not found")
	}

	var rows []ApplicantRow
	err = s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		var err error
		rows, err = repositories.Opportunities.AllApplicants(ctx, opportunityID, filter)
		if err != nil {
			return fmt.Errorf("load applicants: %w", err)
		}
		return audit(ctx, repositories, actorID, "applicants_exported", opportunityID, map[string]string{
			"rows": strconv.Itoa(len(rows)), "status": status,
		}, s.now())
	})
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	_ = writer.Write(exportHeader())
	for _, row := range rows {
		_ = writer.Write([]string{
			csvsafe.Cell(row.FullName),
			csvsafe.Cell(valueOf(row.Email)),
			csvsafe.Cell(valueOf(row.USN)),
			csvsafe.Cell(valueOf(row.DepartmentCode)),
			intOf(row.BatchYear),
			string(row.Mode),
			string(row.Status),
			row.AppliedAt.UTC().Format(time.RFC3339),
			timeOf(row.StatusChangedAt),
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("write csv: %w", err)
	}
	return buffer.Bytes(), nil
}

func valueOf(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func intOf(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}

func timeOf(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
