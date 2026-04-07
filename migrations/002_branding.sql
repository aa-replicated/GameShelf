-- Extended branding: background color, font family, logo blob
ALTER TABLE sites
    ADD COLUMN IF NOT EXISTS background_color VARCHAR(7)   NOT NULL DEFAULT '#F9FAFB',
    ADD COLUMN IF NOT EXISTS font_family      VARCHAR(50)  NOT NULL DEFAULT 'system',
    ADD COLUMN IF NOT EXISTS logo_data        BYTEA,
    ADD COLUMN IF NOT EXISTS logo_content_type VARCHAR(50);
