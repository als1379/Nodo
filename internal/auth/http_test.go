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
		{"valid registration", credentials{"person@example.com", "long-password"}, true, ""},
		{"invalid email", credentials{"invalid", "long-password"}, true, "a valid email is required"},
		{"short registration password", credentials{"person@example.com", "short"}, true, "password must be between 10 and 72 characters"},
		{"empty login password", credentials{"person@example.com", ""}, false, "password is required"},
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
