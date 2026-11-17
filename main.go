package checkhuman

import "github.com/ravikumariitk/gotcha/internal/engine"

// Appearance controls the colors and texture used by a rendered challenge.
// Use ParseAppearance to build one from CSS-style hex colors.
type Appearance = engine.Appearance

// MotionCaptcha is the browser-ready representation of a rendered challenge.
type MotionCaptcha = engine.MotionCaptcha

// ParseAppearance creates a safe appearance configuration. Invalid colors are
// ignored and the default appearance is returned when fewer than two colors
// are supplied.
func ParseAppearance(colors []string, texture string) Appearance {
	return engine.ParseAppearance(colors, texture)
}

// RenderMotion creates a human-readable motion CAPTCHA for code.
func RenderMotion(code, defenseLevel string, appearance Appearance) *MotionCaptcha {
	return engine.GenerateMotionFrames(code, defenseLevel, appearance)
}

// Challenge contains the signed value that should be sent back during
// verification and the media that should be shown to the user. The answer is
// deliberately never included in this type.
type Challenge struct {
	ID        string         `json:"id"`
	Token     string         `json:"token"`
	ExpiresAt int64          `json:"expiresAt"`
	Motion    *MotionCaptcha `json:"motion,omitempty"`
}

// NewChallenge creates a signed challenge and its human-readable motion
// rendering. The code is generated server-side and is never returned.
func NewChallenge(length int, defenseLevel string, appearance Appearance) *Challenge {
	code := GenerateCode(length)
	id, token, expiresAt := IssueCode(code)
	return &Challenge{
		ID:        id,
		Token:     token,
		ExpiresAt: expiresAt,
		Motion:    RenderMotion(code, defenseLevel, appearance),
	}
}

// ConfigureSecret sets the HMAC signing key used by the challenge token.
func ConfigureSecret(raw string) {
	engine.ConfigureSecret(raw)
}

// GenerateCode creates a random numeric code of the requested length.
func GenerateCode(length int) string {
	if length <= 0 {
		length = 5
	}
	return engine.GenerateRandomCode(length)
}

// IssueCode creates a challenge and returns its id, signed token, and expiry timestamp.
func IssueCode(secretCode string) (challengeID string, token string, expiresAt int64) {
	if secretCode == "" {
		secretCode = GenerateCode(5)
	}
	return engine.IssueChallenge(secretCode)
}

// VerifyCode validates a submitted answer against a signed challenge token.
func VerifyCode(token string, answer string) (bool, int64, error) {
	return engine.VerifyAnswer(token, answer)
}
