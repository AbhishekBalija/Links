package events

import (
	"context"
	"fmt"
	"time"
)

// DepartmentEvent is a published Event of a Department, for its HOD's Home.
type DepartmentEvent struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	EventType Type      `json:"event_type"`
	Location  string    `json:"location"`
	StartsAt  time.Time `json:"starts_at"`
}

// UpcomingInDepartment returns a Department's next published Events, soonest
// first, whoever they are for: the HOD runs the Department, so they see its
// Events even when the Audience is only its students.
func (s *Service) UpcomingInDepartment(ctx context.Context, departmentID string, limit int) ([]DepartmentEvent, error) {
	views, err := s.repository.UpcomingInDepartment(ctx, departmentID, limit)
	if err != nil {
		return nil, fmt.Errorf("list department events: %w", err)
	}
	events := make([]DepartmentEvent, 0, len(views))
	for _, view := range views {
		events = append(events, DepartmentEvent{ID: view.ID, Title: view.Title, EventType: view.EventType, Location: view.Location, StartsAt: view.StartsAt})
	}
	return events, nil
}

func (r *GormRepository) UpcomingInDepartment(ctx context.Context, departmentID string, limit int) ([]View, error) {
	var views []View
	err := r.db.WithContext(ctx).Raw(`SELECT `+viewColumns+` `+viewFrom+`
		WHERE e.department_id = ? AND e.status = ? AND e.starts_at > now()
		ORDER BY e.starts_at, e.id LIMIT ?`, departmentID, StatusPublished, limit).Scan(&views).Error
	return views, err
}
