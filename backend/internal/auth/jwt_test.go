package auth

import (
	"testing"
	"time"

	"social_app/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

func TestDLQ_TC_003_AUTH_001_JWTRoundTrip(t *testing.T) {
	cfg := &config.Config{JWTSecret: "dlq-test-secret"}
	before := time.Now()
	token, err := GenerateToken(10000001, "alice", cfg)
	after := time.Now()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if token == "" {
		t.Fatal("GenerateToken() returned an empty token")
	}

	claims, err := ParseToken(token, cfg)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.UserID != 10000001 || claims.Username != "alice" {
		t.Fatalf("claims identity = (%d, %q)", claims.UserID, claims.Username)
	}
	minimum := before.Truncate(time.Second)
	maximum := after.Truncate(time.Second)
	for name, value := range map[string]time.Time{
		"issued-at":  claims.IssuedAt.Time,
		"not-before": claims.NotBefore.Time,
	} {
		if value.Before(minimum) || value.After(maximum) {
			t.Errorf("%s = %v, want within [%v, %v]", name, value, minimum, maximum)
		}
	}
	if got := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time); got != 168*time.Hour {
		t.Errorf("token lifetime = %v, want %v", got, 168*time.Hour)
	}
}

func TestDLQ_TC_004_AUTH_001_JWTRejectsInvalidTokens(t *testing.T) {
	validCfg := &config.Config{JWTSecret: "dlq-test-secret"}
	token, err := GenerateToken(10000001, "alice", validCfg)
	if err != nil {
		t.Fatal(err)
	}
	expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID:   10000001,
		Username: "alice",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}).SignedString([]byte(validCfg.JWTSecret))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		token     string
		cfg       *config.Config
		wantError bool
	}{
		{name: "valid control", token: token, cfg: validCfg},
		{name: "wrong signature", token: token, cfg: &config.Config{JWTSecret: "wrong-secret"}, wantError: true},
		{name: "malformed", token: "not-a-jwt", cfg: validCfg, wantError: true},
		{name: "empty", token: "", cfg: validCfg, wantError: true},
		{name: "expired", token: expired, cfg: validCfg, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			claims, err := ParseToken(test.token, test.cfg)
			if test.wantError && (err == nil || claims != nil) {
				t.Fatalf("ParseToken() = (%v, %v), want nil claims and error", claims, err)
			}
			if !test.wantError && (err != nil || claims == nil) {
				t.Fatalf("ParseToken() = (%v, %v), want valid claims", claims, err)
			}
		})
	}
}
