package departments

import "time"

type CreateDepartmentInput struct {
	Code        string  `json:"code" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	HODUserID   *string `json:"hodUserId"`
}

type UpdateDepartmentInput struct {
	// Code may repeat the Department's own code; any other code is refused,
	// since codes never change (ADR 0021).
	Code        *string `json:"code"`
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	HODUserID   *string `json:"hodUserId"`
}

// RenameDepartmentInput changes only a Department's name. Code, as in
// UpdateDepartmentInput, may only repeat the Department's own code.
type RenameDepartmentInput struct {
	Code *string `json:"code"`
	Name string  `json:"name" binding:"required"`
}

type DepartmentResponse struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	HODUserID   *string   `json:"hodUserId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// PublicDepartment is what the sign-up form needs: no IDs and no names, only
// whether a request goes to an HOD or to the admins (#207).
type PublicDepartment struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	HasHOD bool   `json:"has_hod"`
}

type DepartmentListResponse struct {
	Departments []DepartmentResponse `json:"departments"`
}

// AdminDepartment is one row of the admin's Departments screen. Students and
// Staff are counted the way Home's college panel counts them (#213):
// everyone on the lists, signed in or not yet; staff is every staff role.
type AdminDepartment struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	HOD         *AdminHOD `json:"hod"`
	Students    int       `json:"students"`
	Staff       int       `json:"staff"`
}

// AdminHOD is the Department's HOD, whether or not their profile is public.
type AdminHOD struct {
	UserID   string `json:"user_id"`
	FullName string `json:"full_name"`
	Username string `json:"username"`
}

type AdminDepartmentList struct {
	Departments []AdminDepartment `json:"departments"`
}
