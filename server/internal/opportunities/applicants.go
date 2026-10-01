package opportunities

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	apperrors "github.com/AbhishekBalija/Links/server/internal/shared/errors"
)

// Applicants lists an Opportunity's Applications for placement staff, in the
// order they were made. Opening the list (its first page) is audited.
func (s *Service) Applicants(ctx context.Context, actorID, opportunityID string, query ApplicantQuery) ([]ApplicantResponse, *ListMeta, error) {
	if err := s.requireStaff(ctx, actorID); err != nil {
		return nil, nil, err
	}
	filter, err := s.applicantFilter(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	limit, err := checkLimit(query.Limit)
	if err != nil {
		return nil, nil, err
	}
	after, err := decodeCursor(query.Cursor)
	if err != nil {
		return nil, nil, err
	}
	opportunity, err := s.repository.Find(ctx, opportunityID)
	if err != nil {
		return nil, nil, fmt.Errorf("find opportunity: %w", err)
	}
	if opportunity == nil {
		return nil, nil, apperrors.NewNotFound("opportunity not found")
	}
	rows, err := s.repository.Applicants(ctx, opportunityID, filter, after, limit+1)
	if err != nil {
		return nil, nil, fmt.Errorf("list applicants: %w", err)
	}
	if after == nil {
		err := s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
			return audit(ctx, repositories, actorID, "applicants_viewed", opportunityID, map[string]string{
				"q": strings.TrimSpace(query.Q), "status": query.Status, "department": query.Department, "batch": query.Batch,
			}, s.now())
		})
		if err != nil {
			return nil, nil, err
		}
	}
	meta := &ListMeta{}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		meta.NextCursor = encodeCursor(Cursor{At: last.AppliedAt, ID: last.ID})
	}
	responses := make([]ApplicantResponse, 0, len(rows))
	for _, row := range rows {
		responses = append(responses, toApplicantResponse(row))
	}
	return responses, meta, nil
}

// maxSearchLength caps the applicant search text.
const maxSearchLength = 100

func (s *Service) applicantFilter(ctx context.Context, query ApplicantQuery) (ApplicantFilter, error) {
	var filter ApplicantFilter
	details := map[string]string{}
	if q := strings.TrimSpace(query.Q); q != "" {
		if len([]rune(q)) > maxSearchLength {
			details["q"] = fmt.Sprintf("use at most %d characters", maxSearchLength)
		}
		filter.Search = &q
	}
	if value := strings.TrimSpace(query.Status); value != "" {
		status := ApplicationStatus(value)
		if !validApplicationStatus(status) {
			details["status"] = "use applied, shortlisted, rejected, selected or withdrawn"
		}
		filter.Status = &status
	}
	if code := strings.ToUpper(strings.TrimSpace(query.Department)); code != "" {
		id, err := s.repository.DepartmentIDByCode(ctx, code)
		if err != nil {
			return filter, fmt.Errorf("find department: %w", err)
		}
		if id == nil {
			details["department"] = "no department has this code"
		}
		filter.DepartmentID = id
	}
	if value := strings.TrimSpace(query.Batch); value != "" {
		batch, err := strconv.Atoi(value)
		if err != nil || batch < 2000 || batch > 2100 {
			details["batch"] = "use a batch year between 2000 and 2100"
		}
		filter.BatchYear = &batch
	}
	if len(details) > 0 {
		return filter, apperrors.NewValidation("invalid applicant filter", details)
	}
	return filter, nil
}

// UpdateStatus moves an Application among applied, shortlisted, rejected and
// selected, in any direction so a mistake can be undone. The caller sends
// the status they saw; the row is locked, and a status that has moved on is a
// conflict rather than an overwrite. A withdrawn Application is the
// Student's decision and stays withdrawn.
func (s *Service) UpdateStatus(ctx context.Context, actorID, applicationID string, input StatusInput) (*ApplicantResponse, error) {
	if err := s.requireStaff(ctx, actorID); err != nil {
		return nil, err
	}
	from, to := ApplicationStatus(input.From), ApplicationStatus(input.Status)
	details := map[string]string{}
	if !validApplicationStatus(from) {
		details["from"] = "use the status you saw"
	}
	switch {
	case !staffStatus(to):
		details["status"] = "use applied, shortlisted, rejected or selected"
	case to == from:
		details["status"] = "is already the status"
	}
	if len(details) > 0 {
		return nil, apperrors.NewValidation("invalid status change", details)
	}
	notFound := apperrors.NewNotFound("application not found")
	if _, err := uuid.Parse(applicationID); err != nil {
		return nil, notFound
	}

	now := s.now()
	err := s.unitOfWork.WithinTransaction(ctx, func(repositories Repositories) error {
		application, err := repositories.Opportunities.FindApplicationByIDForUpdate(ctx, applicationID)
		if err != nil {
			return fmt.Errorf("find application: %w", err)
		}
		if application == nil {
			return notFound
		}
		if application.Status == ApplicationWithdrawn {
			return apperrors.NewConflict("the student withdrew this application")
		}
		if application.Status != from {
			return apperrors.NewConflict(fmt.Sprintf("the application is now %s; reload and try again", application.Status))
		}
		application.Status = to
		application.StatusChangedAt = &now
		application.StatusChangedBy = &actorID
		application.UpdatedAt = now
		if err := repositories.Opportunities.UpdateApplication(ctx, application); err != nil {
			return fmt.Errorf("update application: %w", err)
		}
		return auditApplication(ctx, repositories, actorID, "application_status_changed", *application, map[string]string{
			"opportunity_id": application.OpportunityID, "from": string(from), "to": string(to),
		})
	})
	if err != nil {
		return nil, err
	}
	row, err := s.repository.Applicant(ctx, applicationID)
	if err != nil {
		return nil, fmt.Errorf("load application: %w", err)
	}
	if row == nil {
		return nil, notFound
	}
	response := toApplicantResponse(*row)
	return &response, nil
}

// addApplicantCounts fills in each Opportunity's applicant counts, for
// placement staff only.
func (s *Service) addApplicantCounts(ctx context.Context, responses []OpportunityResponse) error {
	ids := make([]string, 0, len(responses))
	for _, response := range responses {
		ids = append(ids, response.ID)
	}
	counts, err := s.repository.ApplicantCounts(ctx, ids)
	if err != nil {
		return fmt.Errorf("count applicants: %w", err)
	}
	for i := range responses {
		count := counts[responses[i].ID]
		responses[i].ApplicantCounts = &count
	}
	return nil
}
