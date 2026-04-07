package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gameshelf/gameshelf/internal/db"
)

const playerCookieName = "gs_player"
const identityTokenExpiry = 30 * 24 * time.Hour

// getOrCreateIdentitySecret returns the HMAC secret used for identity tokens.
// If IDENTITY_SECRET env var is set it takes precedence. Otherwise the secret
// is auto-generated on first call and persisted in the settings table.
func (s *Server) getOrCreateIdentitySecret() (string, error) {
	if s.cfg.IdentitySecret != "" {
		return s.cfg.IdentitySecret, nil
	}
	secret, err := db.GetSetting(s.db, "identity_secret")
	if err != nil {
		return "", fmt.Errorf("get identity secret: %w", err)
	}
	if secret != "" {
		return secret, nil
	}
	// Generate a new 32-byte random secret.
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate identity secret: %w", err)
	}
	secret = base64.RawURLEncoding.EncodeToString(b)
	if err := db.SetSetting(s.db, "identity_secret", secret); err != nil {
		return "", fmt.Errorf("store identity secret: %w", err)
	}
	return secret, nil
}

// SignIdentityToken creates a signed token for the given player name.
//
// Token format (base64url-encoded): "playerName|expiryUnix|hmac-sha256-hex"
//
// The token is intended to be passed as ?gs_identity=<token> in a link from
// the customer's main site so returning players are recognised by GameShelf.
func SignIdentityToken(secret, playerName string) string {
	expiry := strconv.FormatInt(time.Now().Add(identityTokenExpiry).Unix(), 10)
	msg := playerName + "|" + expiry
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	raw := msg + "|" + fmt.Sprintf("%x", mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// verifyIdentityToken validates a token and returns the player name if valid.
func verifyIdentityToken(secret, token string) (string, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", false
	}
	parts := strings.SplitN(string(raw), "|", 3)
	if len(parts) != 3 {
		return "", false
	}
	playerName, expiry, sig := parts[0], parts[1], parts[2]

	exp, err := strconv.ParseInt(expiry, 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return "", false
	}

	msg := playerName + "|" + expiry
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	expected := fmt.Sprintf("%x", mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return "", false
	}
	return playerName, true
}

// setPlayerCookie writes the gs_player cookie to the response.
func setPlayerCookie(w http.ResponseWriter, playerName string) {
	http.SetCookie(w, &http.Cookie{
		Name:     playerCookieName,
		Value:    playerName,
		Path:     "/",
		MaxAge:   30 * 24 * 3600,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: false, // JS also reads/writes it for immediate use
	})
}

// getPlayerFromCookie returns the player name from the gs_player cookie, or "".
func getPlayerFromCookie(r *http.Request) string {
	c, err := r.Cookie(playerCookieName)
	if err != nil {
		return ""
	}
	return c.Value
}
