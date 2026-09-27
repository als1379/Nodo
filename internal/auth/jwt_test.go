package auth

import (
	"errors"
	"testing"
	"time"
)

func TestJWTSignAndVerify(t *testing.T) {
	jwt := NewJWT("a-test-secret-that-is-at-least-32-characters")
	token, err := jwt.Sign("user-id", "session-id", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	claims, err := jwt.Verify(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "user-id" || claims.SessionID != "session-id" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if _, err := jwt.Verify(token + "changed"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token, got %v", err)
	}
}

func TestJWTRejectsExpiredToken(t *testing.T) {
	jwt := NewJWT("a-test-secret-that-is-at-least-32-characters")
	token, err := jwt.Sign("user-id", "session-id", time.Now().Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jwt.Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token, got %v", err)
	}
}
