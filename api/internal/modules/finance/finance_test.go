package finance

import "testing"

func TestUserName(t *testing.T) {
	cases := []struct {
		name, email, phone, want string
	}{
		{"email local part", "gamehub@salehcard.local", "+9613000000", "gamehub"},
		{"email without at", "weird", "", "weird"},
		{"phone fallback", "", "+9613000000", "+9613000000"},
		{"orphaned row", "", "", "Unknown"},
		{"email preferred over phone", "ali@x.com", "+961", "ali"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := userName(c.email, c.phone); got != c.want {
				t.Errorf("userName(%q, %q) = %q, want %q", c.email, c.phone, got, c.want)
			}
		})
	}
}
