package db

import "time"

type Site struct {
	ID               int
	Name             string
	PrimaryColor     string
	SecondaryColor   string
	BackgroundColor  string
	FontFamily       string
	HasLogo          bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Player struct {
	ID          int
	DisplayName string
	CreatedAt   time.Time
}

type Game struct {
	ID          int
	Slug        string
	Name        string
	Description string
	Enabled     bool
	MinPlayers  int
	MaxPlayers  int
	CreatedAt   time.Time
}

type Score struct {
	ID         int
	PlayerID   int
	PlayerName string
	GameSlug   string
	Score      int
	PlayedAt   time.Time
}
