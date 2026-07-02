package auth

import "testing"

func TestActorLabel(t *testing.T) {
	cases := []struct {
		name  string
		claim *Claims
		want  string
	}{
		{"prefers email", &Claims{UserID: "u1", Email: "admin@x.io", Phone: "+9611"}, "admin@x.io"},
		{"falls back to phone", &Claims{UserID: "u1", Email: "", Phone: "+9611"}, "+9611"},
		{"falls back to user id", &Claims{UserID: "u1", Email: "", Phone: ""}, "u1"},
		{"nil is empty", nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ActorLabel(tc.claim); got != tc.want {
				t.Fatalf("ActorLabel = %q, want %q", got, tc.want)
			}
		})
	}
}
