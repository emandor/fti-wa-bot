package config

import "testing"

func TestLoadFromEnv_DefaultPortAndRequiredAuth(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("AUTH_TOKEN", "secret")
	t.Setenv("GROUP_JID", "123@g.us")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv returned error: %v", err)
	}

	if cfg.Port != 5000 {
		t.Fatalf("expected default port 5000, got %d", cfg.Port)
	}

	if cfg.AuthToken != "secret" {
		t.Fatalf("expected auth token to be set")
	}

	if cfg.GroupJID != "123@g.us" {
		t.Fatalf("expected group jid to be set")
	}

	if cfg.AuthDBDSN == "" || cfg.LogsDBDSN == "" {
		t.Fatalf("expected db DSN defaults to be set")
	}
}

func TestLoadFromEnv_EmptyAuthTokenFails(t *testing.T) {
	t.Setenv("AUTH_TOKEN", "  ")

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("expected error for empty AUTH_TOKEN")
	}

	if err != ErrEmptyAuthToken {
		t.Fatalf("expected ErrEmptyAuthToken, got %v", err)
	}
}

func TestLoadFromEnv_InvalidPortFails(t *testing.T) {
	t.Setenv("PORT", "abc")
	t.Setenv("AUTH_TOKEN", "secret")

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("expected invalid port error")
	}
}
