package departments

import "time"

type CreateDepartmentInput struct {
	Code        string  `json:"code" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	HODUserID   *string `json:"hodUserId"`
}

type UpdateDepartmentInput struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	HODUserID   *string `json:"hodUserId"`
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

type DepartmentListResponse struct {
	Departments []DepartmentResponse `json:"departments"`
}
