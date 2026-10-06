package dashboard

import (
	"context"
	"fmt"

	"github.com/AbhishekBalija/Links/server/internal/announcements"
	"github.com/AbhishekBalija/Links/server/internal/auth"
	"github.com/AbhishekBalija/Links/server/internal/departments"
	"github.com/AbhishekBalija/Links/server/internal/directory"
	"github.com/AbhishekBalija/Links/server/internal/events"
)

// eventsOnDepartmentHome is how many upcoming Department events an HOD sees.
const eventsOnDepartmentHome = 3

// Directory is what the dashboard needs from the directory: a Department's
// counts and HOD.
type Directory interface {
	Overview(ctx context.Context, viewerID, code string) (*directory.Overview, error)
}

// Departments lists every Department for the college panel.
type Departments interface {
	ListPublic(ctx context.Context) ([]departments.PublicDepartment, error)
}

// Access is what the dashboard needs from auth: the Access requests waiting.
type Access interface {
	AccessSummary(ctx context.Context, actorID string) (*auth.AccessSummary, error)
	ListsSummary(ctx context.Context, actorID string) (*auth.ListsSummary, error)
	NewRole(ctx context.Context, userID string) (*auth.NewRole, error)
}

// DepartmentSection is an HOD's own Department on Home.
type DepartmentSection struct {
	Code            string                   `json:"code"`
	Name            string                   `json:"name"`
	Students        int                      `json:"students"`
	Staff           int                      `json:"staff"`
	StudentsByBatch []directory.BatchCount   `json:"students_by_batch"`
	UpcomingEvents  []events.DepartmentEvent `json:"upcoming_events"`
}

// CollegeDepartment is one row of the principal's and admins' college panel.
type CollegeDepartment struct {
	Code     string   `json:"code"`
	Name     string   `json:"name"`
	Students int      `json:"students"`
	Staff    int      `json:"staff"`
	HOD      *HODName `json:"hod"`
}

type HODName struct {
	FullName string `json:"full_name"`
	Username string `json:"username"`
}

type CollegeSection struct {
	Departments           []CollegeDepartment `json:"departments"`
	DepartmentsWithoutHOD int                 `json:"departments_without_hod"`
}

// department builds the HOD's Department section, or nil for anyone who
// isn't an HOD.
func (s *Service) department(ctx context.Context, userID string, grants []announcements.Grant) (*DepartmentSection, error) {
	departmentID := ""
	for _, grant := range grants {
		if grant.Role == auth.RoleHOD && grant.DepartmentID != "" {
			departmentID = grant.DepartmentID
			break
		}
	}
	if departmentID == "" {
		return nil, nil
	}
	department, err := s.repository.Department(ctx, departmentID)
	if err != nil || department == nil {
		return nil, err
	}
	overview, err := s.directory.Overview(ctx, userID, department.Code)
	if err != nil {
		return nil, fmt.Errorf("department overview: %w", err)
	}
	upcoming, err := s.events.UpcomingInDepartment(ctx, departmentID, eventsOnDepartmentHome)
	if err != nil {
		return nil, err
	}
	return &DepartmentSection{
		Code:            department.Code,
		Name:            department.Name,
		Students:        overview.Counts.Students,
		Staff:           overview.Counts.Staff,
		StudentsByBatch: overview.Counts.StudentsByBatch,
		UpcomingEvents:  upcoming,
	}, nil
}

// college builds every Department's row for the principal and admins, or nil
// for anyone else.
func (s *Service) college(ctx context.Context, userID string, roles []string) (*CollegeSection, error) {
	if !hasAny(roles, string(auth.RolePrincipal), string(auth.RoleAdmin)) {
		return nil, nil
	}
	list, err := s.departments.ListPublic(ctx)
	if err != nil {
		return nil, err
	}
	section := &CollegeSection{Departments: make([]CollegeDepartment, 0, len(list))}
	for _, d := range list {
		overview, err := s.directory.Overview(ctx, userID, d.Code)
		if err != nil {
			return nil, fmt.Errorf("overview of %s: %w", d.Code, err)
		}
		row := CollegeDepartment{Code: d.Code, Name: d.Name, Students: overview.Counts.Students, Staff: overview.Counts.Staff}
		if overview.HOD != nil {
			row.HOD = &HODName{FullName: overview.HOD.FullName, Username: overview.HOD.Username}
		}
		if row.HOD == nil {
			section.DepartmentsWithoutHOD++
		}
		section.Departments = append(section.Departments, row)
	}
	return section, nil
}

func hasAny(roles []string, wanted ...string) bool {
	for _, role := range roles {
		for _, w := range wanted {
			if role == w {
				return true
			}
		}
	}
	return false
}
