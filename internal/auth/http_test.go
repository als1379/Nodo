package auth

import (
	"net/http"
	"testing"
)

func TestValidateCredentials(t *testing.T) {
	tests := []struct {
		name   string
		input  credentials
		strong bool
		want   string
	}{
		{"valid registration", credentials{Email: "person@example.com", Password: "long-password"}, true, ""},
		{"invalid email", credentials{Email: "invalid", Password: "long-password"}, true, "a valid email is required"},
		{"short registration password", credentials{Email: "person@example.com", Password: "short"}, true, "password must be between 10 and 72 characters"},
		{"valid username login", credentials{Login: "admin", Password: "ali"}, false, ""},
		{"empty login password", credentials{Login: "admin"}, false, "password is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateCredentials(tt.input, tt.strong); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBearerToken(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer abc123")
	if got := bearerToken(r); got != "abc123" {
		t.Fatalf("got %q", got)
	}
}
