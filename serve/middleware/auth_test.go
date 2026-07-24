package middleware

import (
	"net/url"
	"sync"
	"testing"

	"github.com/HuolalaTech/page-spy-api/config"
)

func TestRedactRequestURI(t *testing.T) {
	raw := "/api/v1/ws/room/join?address=room.local&secret=first&secret=second&token=jwt-value&name=device"
	redacted := redactRequestURI(raw)

	parsed, err := url.Parse(redacted)
	if err != nil {
		t.Fatalf("parse redacted URI: %v", err)
	}

	if got := parsed.Query().Get("address"); got != "room.local" {
		t.Fatalf("address changed: %q", got)
	}
	if got := parsed.Query().Get("name"); got != "device" {
		t.Fatalf("name changed: %q", got)
	}
	for _, key := range []string{"secret", "token"} {
		for _, value := range parsed.Query()[key] {
			if value != "***" {
				t.Fatalf("%s was not redacted: %q", key, value)
			}
		}
	}
}

func TestRedactInvalidRequestURI(t *testing.T) {
	if got := redactRequestURI("/api?secret=%zz"); got != "/api" {
		t.Fatalf("invalid URI should not be logged verbatim: %q", got)
	}
}

func TestVerifyPassword(t *testing.T) {
	cfg := &config.Config{AuthConfig: &config.AuthConfig{Password: "correct-password"}}
	if !VerifyPassword(cfg, "correct-password") {
		t.Fatal("correct password was rejected")
	}
	if VerifyPassword(cfg, "wrong-password") {
		t.Fatal("wrong password was accepted")
	}
	if VerifyPassword(&config.Config{}, "") {
		t.Fatal("missing auth config must not accept a password")
	}
}

func TestConcurrentJWTInitializationUsesOneSecret(t *testing.T) {
	jwtSecretMu.Lock()
	previousSecret := append([]byte(nil), jwtSecret...)
	jwtSecret = nil
	jwtSecretMu.Unlock()
	defer func() {
		jwtSecretMu.Lock()
		jwtSecret = previousSecret
		jwtSecretMu.Unlock()
	}()

	cfg := &config.Config{AuthConfig: &config.AuthConfig{
		Password:        "password",
		JwtSecret:       "test-secret",
		TokenExpiration: 1,
	}}

	const tokenCount = 64
	tokens := make(chan string, tokenCount)
	errs := make(chan error, tokenCount)
	var wg sync.WaitGroup
	for i := 0; i < tokenCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, _, err := GenerateToken(cfg)
			if err != nil {
				errs <- err
				return
			}
			tokens <- token
		}()
	}
	wg.Wait()
	close(tokens)
	close(errs)

	for err := range errs {
		t.Fatalf("generate token: %v", err)
	}
	for token := range tokens {
		if _, err := ParseToken(token); err != nil {
			t.Fatalf("token generated during concurrent initialization is invalid: %v", err)
		}
	}
}
