// Package authorwork is the shape of an author's own work that needs
// attention, shared by the modules that have authors (Announcements and
// Events) and by Home, which lists both.
package authorwork

import "time"

// PrincipalOrAdmin names who decides when no Department HOD does.
const PrincipalOrAdmin = "Principal or admin"

// Kind is the sort of thing an item is.
type Kind string

const (
	KindAnnouncement Kind = "announcement"
	KindEvent        Kind = "event"
)

// SentBack is something a reviewer sent back to its author to fix, with the
// reviewer's note. IsEdit marks an edit to a published Announcement.
type SentBack struct {
	Kind       Kind      `json:"kind"`
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Note       *string   `json:"note"`
	SentBackBy string    `json:"sent_back_by"`
	SentBackAt time.Time `json:"sent_back_at"`
	IsEdit     bool      `json:"is_edit"`
}

// Waiting is something at a reviewer, and since when. IsEdit marks an edit
// to a published Announcement.
type Waiting struct {
	Kind      Kind      `json:"kind"`
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	WaitingOn string    `json:"waiting_on"`
	Since     time.Time `json:"since"`
	IsEdit    bool      `json:"is_edit"`
}
