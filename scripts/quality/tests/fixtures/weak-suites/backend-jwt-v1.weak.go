package auth

import (
	"testing"

	"social_app/internal/config"
)

func TestDLQ_TC_029_WeakJWTBaseline(t *testing.T) {
	token, err := GenerateToken(10000001, "alice", &config.Config{JWTSecret: "dlq-test-secret"})
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if token == "" {
		t.Fatal("GenerateToken() returned an empty token")
	}
}
