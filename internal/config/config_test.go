package config

import "testing"

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("SITE_NAME", "")
	t.Setenv("ADMIN_SECRET", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("LOCAL_DEV", "")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("Port default: got %q, want %q", cfg.Port, "8080")
	}
	if cfg.SiteName != "GameShelf" {
		t.Errorf("SiteName default: got %q, want %q", cfg.SiteName, "GameShelf")
	}
	if cfg.AdminSecret != "changeme" {
		t.Errorf("AdminSecret default: got %q, want %q", cfg.AdminSecret, "changeme")
	}
	if cfg.DatabaseURL != "" {
		t.Errorf("DatabaseURL should be empty by default, got %q", cfg.DatabaseURL)
	}
	if cfg.RedisURL != "" {
		t.Errorf("RedisURL should be empty by default, got %q", cfg.RedisURL)
	}
	if cfg.LocalDev {
		t.Errorf("LocalDev should be false by default")
	}
}

func TestLoad_LocalDev(t *testing.T) {
	t.Setenv("LOCAL_DEV", "true")
	if cfg := Load(); !cfg.LocalDev {
		t.Error("LocalDev should be true when LOCAL_DEV=true")
	}

	t.Setenv("LOCAL_DEV", "false")
	if cfg := Load(); cfg.LocalDev {
		t.Error("LocalDev should be false when LOCAL_DEV=false")
	}

	t.Setenv("LOCAL_DEV", "1") // only "true" activates it
	if cfg := Load(); cfg.LocalDev {
		t.Error("LocalDev should be false when LOCAL_DEV=1")
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("SITE_NAME", "MyNewsPaper")
	t.Setenv("ADMIN_SECRET", "supersecret")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("REDIS_URL", "redis://localhost:6379")

	cfg := Load()

	if cfg.Port != "9090" {
		t.Errorf("Port: got %q, want %q", cfg.Port, "9090")
	}
	if cfg.SiteName != "MyNewsPaper" {
		t.Errorf("SiteName: got %q, want %q", cfg.SiteName, "MyNewsPaper")
	}
	if cfg.AdminSecret != "supersecret" {
		t.Errorf("AdminSecret: got %q, want %q", cfg.AdminSecret, "supersecret")
	}
	if cfg.DatabaseURL != "postgres://localhost/test" {
		t.Errorf("DatabaseURL: got %q, want %q", cfg.DatabaseURL, "postgres://localhost/test")
	}
	if cfg.RedisURL != "redis://localhost:6379" {
		t.Errorf("RedisURL: got %q, want %q", cfg.RedisURL, "redis://localhost:6379")
	}
}
