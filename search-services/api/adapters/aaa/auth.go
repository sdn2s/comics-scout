package aaa

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

const (
	secretKey        = "something secret here"
	adminRole        = "superuser"
	envAdminUser     = "ADMIN_USER"
	envAdminPassword = "ADMIN_PASSWORD"

	defaultAdminUser     = "admin"
	defaultAdminPassword = "password"
)

type AAA struct {
	users    map[string]string
	tokenTTL time.Duration
	log      *slog.Logger
}

func New(tokenTTL time.Duration, log *slog.Logger) (AAA, error) {
	if log == nil {
		log = slog.Default()
	}

	users := map[string]string{
		defaultAdminUser: defaultAdminPassword,
	}

	if u, okU := os.LookupEnv(envAdminUser); okU {
		if p, okP := os.LookupEnv(envAdminPassword); okP {
			users[u] = p
			log.Info("admin credentials loaded from env",
				"user", u,
				"vars", envAdminUser+"/"+envAdminPassword,
			)
		}
	}

	if tokenTTL <= 0 {
		tokenTTL = 2 * time.Minute
	}

	return AAA{
		users:    users,
		tokenTTL: tokenTTL,
		log:      log,
	}, nil
}

func (a AAA) Login(name, password string) (string, error) {
	expected, ok := a.users[name]
	if !ok || expected != password {
		return "", errors.New("invalid credentials")
	}

	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   adminRole,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(a.tokenTTL)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString([]byte(secretKey))
	if err != nil {
		a.log.Error("failed to sign token", "error", err)
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signed, nil
}

func (a AAA) Verify(tokenString string) error {
	if tokenString == "" {
		return errors.New("empty token")
	}

	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secretKey), nil
		},
	)
	if err != nil {
		a.log.Warn("failed to parse token", "error", err)
		return err
	}

	if !token.Valid {
		return errors.New("invalid token")
	}

	if claims.Subject != adminRole {
		return errors.New("forbidden")
	}

	if claims.ExpiresAt == nil {
		return errors.New("token without exp")
	}
	if time.Now().After(claims.ExpiresAt.Time) {
		return errors.New("token expired")
	}

	return nil
}
