package directory

// DepartmentRef names a member's Department.
type DepartmentRef struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// Entry is one member as the directory shows them. It never carries a USN.
type Entry struct {
	Username   string         `json:"username"`
	FullName   string         `json:"full_name"`
	Headline   *string        `json:"headline"`
	AvatarURL  *string        `json:"avatar_url"`
	Roles      []string       `json:"roles"`
	Department *DepartmentRef `json:"department"`
	BatchYear  *int           `json:"batch_year,omitempty"`
	Email      *string        `json:"email,omitempty"`
	Phone      *string        `json:"phone,omitempty"`
}

// ListQuery is the directory's query string, parsed as text so the service
// can report each bad field.
type ListQuery struct {
	Department string
	Role       string
	Batch      string
	Q          string
	Limit      string
	Cursor     string
}

type ListMeta struct {
	NextCursor string `json:"next_cursor,omitempty"`
	// Total is how many members match the filters (and search) in all.
	Total int `json:"total"`
}

// Overview is a Department's page: who leads and teaches it, and how many
// students it has.
type Overview struct {
	Department OverviewDepartment `json:"department"`
	HOD        *Entry             `json:"hod"`
	Counts     OverviewCounts     `json:"counts"`
	Staff      []Entry            `json:"staff"`
}

type OverviewDepartment struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type OverviewCounts struct {
	Students int `json:"students"`
	Faculty  int `json:"faculty"`
	// Staff counts every staff role (HOD, placement officer, faculty) once.
	Staff           int          `json:"staff"`
	StudentsByBatch []BatchCount `json:"students_by_batch"`
}
