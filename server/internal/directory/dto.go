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
}
