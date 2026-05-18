package aaa

import (
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

func TestNew_UsesEnvCredentialsAndDefaults(t *testing.T) {
	t.Setenv(envAdminUser, "root")
	t.Setenv(envAdminPassword, "toor")

	auth, err := New(0, nil)
	if err != nil {
		t.Fatalf("new auth failed: %v", err)
	}

	token, err := auth.Login("root", "toor")
	if err != nil {
		t.Fatalf("expected login to succeed: %v", err)
	}
	if token == "" {
		t.Fatalf("expected non-empty token")
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	auth, _ := New(time.Minute, nil)

	if _, err := auth.Login("bad", "creds"); err == nil {
		t.Fatalf("expected error for invalid credentials")
	}
}

func TestVerify(t *testing.T) {
	auth, _ := New(time.Minute, nil)
	token, err := auth.Login(defaultAdminUser, defaultAdminPassword)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if err := auth.Verify(token); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
}

func TestVerify_ExpiredToken(t *testing.T) {
	auth, _ := New(time.Minute, nil)

	claims := jwt.RegisteredClaims{
		Subject:   adminRole,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secretKey))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	if err := auth.Verify(signed); err == nil {
		t.Fatalf("expected expired token error")
	}
}
