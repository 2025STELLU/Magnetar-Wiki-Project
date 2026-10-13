package auth

import (
	"context"
	"os"
	"testing"

	"github.com/2025STELLU/MagnetarWiki/internal/db"
	"github.com/2025STELLU/MagnetarWiki/internal/session"
)

const (
	envSecretKey  = "MAGNETAR_TEST_SECRET_KEY"
	envSessionKey = "MAGNETAR_TEST_SESSION_KEY"
	envSessionUse = "MAGNETAR_TEST_SESSION_USER"
)

func TestResolveSessionWrittenByDjango(t *testing.T) {
	dsn := os.Getenv(db.EnvDSN)
	secretKey := os.Getenv(envSecretKey)
	sessionKey := os.Getenv(envSessionKey)
	wantUser := os.Getenv(envSessionUse)
	if dsn == "" || secretKey == "" || sessionKey == "" || wantUser == "" {
		t.Skipf("set %s, %s, %s and %s to run this", db.EnvDSN, envSecretKey, envSessionKey, envSessionUse)
	}

	ctx := context.Background()
	conn, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open(dsn) err = %v, want nil", err)
	}
	t.Cleanup(conn.Close)

	f := newFixture(t)
	f.resolver = NewResolver(session.New(secretKey), conn, conn, quietLog())

	got := f.resolver.resolve(f.request(t, sessionKey))
	if got == nil {
		t.Fatalf("resolve(%s) = nil, want %q", envSessionKey, wantUser)
	}
	if got.Username != wantUser {
		t.Errorf("resolve(%s).Username = %q, want %q", envSessionKey, got.Username, wantUser)
	}
}

func TestResolveRejectsATamperedLiveSessionKey(t *testing.T) {
	dsn := os.Getenv(db.EnvDSN)
	secretKey := os.Getenv(envSecretKey)
	sessionKey := os.Getenv(envSessionKey)
	if dsn == "" || secretKey == "" || sessionKey == "" {
		t.Skipf("set %s, %s and %s to run this", db.EnvDSN, envSecretKey, envSessionKey)
	}

	ctx := context.Background()
	conn, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open(dsn) err = %v, want nil", err)
	}
	t.Cleanup(conn.Close)

	f := newFixture(t)
	f.resolver = NewResolver(session.New(secretKey), conn, conn, quietLog())

	tampered := "z" + sessionKey[1:]
	if got := f.resolver.resolve(f.request(t, tampered)); got != nil {
		t.Errorf("resolve(altered session key) = %v, want nil", got)
	}
}
