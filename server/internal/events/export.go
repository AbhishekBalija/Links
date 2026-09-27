package events

// Participant export: the Event's organisers download who answered as CSV.

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

var exportHeader = []string{"full_name", "email", "usn", "batch_year", "department", "rsvp_status", "responded_at"}

// Export returns the Event's participants as CSV to its organisers (the
// proposer, the Department's HOD, the principal and admins), and records the
// export in the audit log in the same transaction. Others who can see the
// Event are forbidden; anyone else gets not found.
func (s *Service) Export(ctx context.Context, actorID, id string) ([]byte, error) {
	organiser, err := s.access(ctx, actorID, id)
	if err != nil {
		return nil, err
	}
	if !organiser {
		return nil, apperrors.NewForbidden("only the event's organisers can export its participants")
	}

	var rows []ExportRow
	now := s.now()
	err = s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		var err error
		rows, err = repositories.Events.ExportRows(ctx, id)
		if err != nil {
			return fmt.Errorf("load participants: %w", err)
		}
		return audit(ctx, repositories, actorID, "event_participants_exported", id, map[string]string{"rows": strconv.Itoa(len(rows))}, now)
	})
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	_ = writer.Write(exportHeader)
	for _, row := range rows {
		batch := ""
		if row.BatchYear != nil {
			batch = strconv.Itoa(*row.BatchYear)
		}
		_ = writer.Write([]string{
			safeCell(row.FullName),
			safeCell(valueOf(row.Email)),
			safeCell(valueOf(row.USN)),
			batch,
			safeCell(valueOf(row.DepartmentCode)),
			string(row.Status),
			row.RespondedAt.UTC().Format(time.RFC3339),
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("write csv: %w", err)
	}
	return buffer.Bytes(), nil
}

// safeCell stops a spreadsheet from running a cell as a formula: text that
// starts with =, +, -, @ or a control character gets a leading quote.
func safeCell(value string) string {
	if value != "" && strings.ContainsRune("=+-@\t\r", rune(value[0])) {
		return "'" + value
	}
	return value
}

func valueOf(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
