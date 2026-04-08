package config

import "os"

type Config struct {
	DatabaseURL    string
	RedisURL       string
	AdminSecret    string
	Port           string
	SiteName       string
	IdentitySecret string // optional; auto-generated and stored in DB if empty
	SDKServiceURL  string // URL of Replicated SDK sidecar, e.g. http://localhost:3000
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	siteName := os.Getenv("SITE_NAME")
	if siteName == "" {
		siteName = "GameShelf"
	}
	adminSecret := os.Getenv("ADMIN_SECRET")
	if adminSecret == "" {
		adminSecret = "changeme"
	}
	return Config{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		RedisURL:       os.Getenv("REDIS_URL"),
		AdminSecret:    adminSecret,
		Port:           port,
		SiteName:       siteName,
		IdentitySecret: os.Getenv("IDENTITY_SECRET"),
		SDKServiceURL:  os.Getenv("SDK_SERVICE_URL"),
	}
}
