-- Sites: tenant branding
CREATE TABLE IF NOT EXISTS sites (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(255) NOT NULL DEFAULT 'GameShelf',
    logo_url        TEXT,
    primary_color   VARCHAR(7)   NOT NULL DEFAULT '#3B82F6',
    secondary_color VARCHAR(7)   NOT NULL DEFAULT '#1E40AF',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Games: registry of available games
CREATE TABLE IF NOT EXISTS games (
    id          SERIAL PRIMARY KEY,
    slug        VARCHAR(100) UNIQUE NOT NULL,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    enabled     BOOLEAN      NOT NULL DEFAULT true,
    min_players INT          NOT NULL DEFAULT 1,
    max_players INT          NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Players: reader accounts (display names only)
CREATE TABLE IF NOT EXISTS players (
    id           SERIAL PRIMARY KEY,
    display_name VARCHAR(255) NOT NULL UNIQUE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Scores: game results
CREATE TABLE IF NOT EXISTS scores (
    id        SERIAL PRIMARY KEY,
    player_id INT         NOT NULL REFERENCES players(id),
    game_slug VARCHAR(100) NOT NULL REFERENCES games(slug),
    score     INT         NOT NULL,
    played_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Settings: app-level key/value config
CREATE TABLE IF NOT EXISTS settings (
    key        VARCHAR(255) PRIMARY KEY,
    value      TEXT        NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_scores_game_slug ON scores(game_slug);
CREATE INDEX IF NOT EXISTS idx_scores_played_at ON scores(played_at DESC);
CREATE INDEX IF NOT EXISTS idx_scores_player_id ON scores(player_id);

-- Seed: default site branding
INSERT INTO sites (name, primary_color, secondary_color)
SELECT 'GameShelf', '#3B82F6', '#1E40AF'
WHERE NOT EXISTS (SELECT 1 FROM sites);

-- Seed: built-in games
INSERT INTO games (slug, name, description, enabled, min_players, max_players) VALUES
    ('word-search',  'Word Search',     'Find hidden words in a letter grid',          true, 1, 1),
    ('anagram',      'Anagram',         'Unscramble letters to form words',            true, 1, 1),
    ('snake',        'Snake',           'Classic snake — eat food, grow longer!',      true, 1, 1),
    ('snood',        'Bubble Shooter',  'Aim and fire to match 3 bubbles and clear the board', true, 1, 1)
ON CONFLICT (slug) DO NOTHING;
