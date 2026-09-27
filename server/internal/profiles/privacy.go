package profiles

// Privacy holds the settings that decide who sees a profile and its contact
// details. The public profile endpoint and the member directory both apply
// these rules, so the two can't drift apart.
type Privacy struct {
	OwnerID              string
	PublicProfileEnabled bool
	ShowEmail            bool
	ShowPhone            bool
}

// Privacy returns the profile's privacy settings.
func (p *Profile) Privacy() Privacy {
	return Privacy{
		OwnerID:              p.UserID,
		PublicProfileEnabled: p.PublicProfileEnabled,
		ShowEmail:            p.ShowEmail,
		ShowPhone:            p.ShowPhone,
	}
}

// VisibleTo reports whether the viewer may see the profile at all: anyone
// when it is public, only its owner otherwise. A nil viewer is anonymous.
func (p Privacy) VisibleTo(viewerID *string) bool {
	return p.PublicProfileEnabled || p.isOwner(viewerID)
}

// ShowsEmailTo reports whether the viewer may see the email address: its
// owner always, anyone else only when the owner opted in.
func (p Privacy) ShowsEmailTo(viewerID *string) bool {
	return p.ShowEmail || p.isOwner(viewerID)
}

// ShowsPhoneTo is ShowsEmailTo for the phone number.
func (p Privacy) ShowsPhoneTo(viewerID *string) bool {
	return p.ShowPhone || p.isOwner(viewerID)
}

func (p Privacy) isOwner(viewerID *string) bool {
	return viewerID != nil && *viewerID == p.OwnerID
}
