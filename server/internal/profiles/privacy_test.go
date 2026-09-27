package profiles

import "testing"

func TestPrivacyRules(t *testing.T) {
	owner, other := "owner-id", "other-id"
	tests := []struct {
		name                  string
		privacy               Privacy
		viewer                *string
		visible, email, phone bool
	}{
		{"public, anonymous", Privacy{OwnerID: owner, PublicProfileEnabled: true}, nil, true, false, false},
		{"public, other member", Privacy{OwnerID: owner, PublicProfileEnabled: true}, &other, true, false, false},
		{"hidden, other member", Privacy{OwnerID: owner}, &other, false, false, false},
		{"hidden, owner", Privacy{OwnerID: owner}, &owner, true, true, true},
		{"opted in", Privacy{OwnerID: owner, PublicProfileEnabled: true, ShowEmail: true, ShowPhone: true}, &other, true, true, true},
		{"email only", Privacy{OwnerID: owner, PublicProfileEnabled: true, ShowEmail: true}, nil, true, true, false},
	}
	for _, test := range tests {
		if got := test.privacy.VisibleTo(test.viewer); got != test.visible {
			t.Errorf("%s: VisibleTo = %v, want %v", test.name, got, test.visible)
		}
		if got := test.privacy.ShowsEmailTo(test.viewer); got != test.email {
			t.Errorf("%s: ShowsEmailTo = %v, want %v", test.name, got, test.email)
		}
		if got := test.privacy.ShowsPhoneTo(test.viewer); got != test.phone {
			t.Errorf("%s: ShowsPhoneTo = %v, want %v", test.name, got, test.phone)
		}
	}
}
