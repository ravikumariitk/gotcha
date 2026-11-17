package engine

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	secretKey      []byte
	tokenTTL       = 3 * time.Minute
	activeMu       sync.RWMutex
	activeTokens   = make(map[string]*ChallengeRecord)
	consumedMu     sync.RWMutex
	consumedTokens = make(map[string]bool)
)

type ChallengeRecord struct {
	AnswerHash string
	ExpiresAt  int64
	CreatedAt  int64
	Attempts   int
}

// GenerateRandomCode returns a cryptographically secure N-digit number string
func GenerateRandomCode(length int) string {
	if length <= 0 {
		length = 5
	}
	digits := "0123456789"
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		b[i] = digits[num.Int64()]
	}
	return string(b)
}

// IssueChallenge creates an HMAC signed token with expiration and anti-replay state
func IssueChallenge(code string) (challengeID string, token string, expiresAt int64) {
	randBytes := make([]byte, 12)
	if _, err := rand.Read(randBytes); err != nil {
		return "", "", 0
	}
	challengeID = "ch_" + hex.EncodeToString(randBytes)

	now := time.Now().UnixMilli()
	expiresAt = now + tokenTTL.Milliseconds()

	// Hash answer with challengeID and timestamp
	secret := configuredSecret()
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(fmt.Sprintf("%s:%s:%d", code, challengeID, now)))
	answerHash := hex.EncodeToString(h.Sum(nil))

	// Payload & Signature
	payload := fmt.Sprintf("%s.%d.%d.%s", challengeID, now, expiresAt, answerHash)
	sigH := hmac.New(sha256.New, secret)
	sigH.Write([]byte(payload))
	signature := hex.EncodeToString(sigH.Sum(nil))

	token = fmt.Sprintf("%s.%s", payload, signature)

	activeMu.Lock()
	activeTokens[challengeID] = &ChallengeRecord{
		AnswerHash: answerHash,
		ExpiresAt:  expiresAt,
		CreatedAt:  now,
		Attempts:   0,
	}
	activeMu.Unlock()

	return challengeID, token, expiresAt
}

// VerifyAnswer checks user's code against the HMAC token
func VerifyAnswer(token string, userAnswer string) (bool, int64, error) {
	if token == "" || userAnswer == "" {
		return false, 0, errors.New("missing token or answer")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 5 {
		return false, 0, errors.New("malformed token")
	}

	challengeID := parts[0]
	tsStr := parts[1]
	expStr := parts[2]
	expectedHash := parts[3]
	sig := parts[4]

	now := time.Now().UnixMilli()
	ts, tsErr := strconv.ParseInt(tsStr, 10, 64)
	exp, expErr := strconv.ParseInt(expStr, 10, 64)
	if tsErr != nil || expErr != nil {
		return false, 0, errors.New("malformed token timestamps")
	}

	// 1. Signature check
	payload := fmt.Sprintf("%s.%s.%s.%s", challengeID, tsStr, expStr, expectedHash)
	secret := configuredSecret()
	sigH := hmac.New(sha256.New, secret)
	sigH.Write([]byte(payload))
	expectedSig := hex.EncodeToString(sigH.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return false, 0, errors.New("invalid token signature")
	}

	// 2. Expiration check
	if now > exp {
		return false, 0, errors.New("challenge expired. Please refresh")
	}

	// 3. Replay check
	consumedMu.RLock()
	if consumedTokens[token] {
		consumedMu.RUnlock()
		return false, 0, errors.New("challenge token already consumed")
	}
	consumedMu.RUnlock()

	// 4. Record lookup
	activeMu.Lock()
	record, exists := activeTokens[challengeID]
	if !exists {
		activeMu.Unlock()
		return false, 0, errors.New("challenge session not found or already verified")
	}

	record.Attempts++
	attemptsExhausted := record.Attempts >= 5
	activeMu.Unlock()

	// 5. Answer hash check
	cleanAnswer := strings.TrimSpace(userAnswer)
	ansH := hmac.New(sha256.New, secret)
	ansH.Write([]byte(fmt.Sprintf("%s:%s:%d", cleanAnswer, challengeID, ts)))
	computedHash := hex.EncodeToString(ansH.Sum(nil))

	if hmac.Equal([]byte(computedHash), []byte(expectedHash)) {
		activeMu.Lock()
		delete(activeTokens, challengeID)
		activeMu.Unlock()

		consumedMu.Lock()
		consumedTokens[token] = true
		consumedMu.Unlock()

		return true, now - ts, nil
	}

	if attemptsExhausted {
		activeMu.Lock()
		delete(activeTokens, challengeID)
		activeMu.Unlock()

		consumedMu.Lock()
		consumedTokens[token] = true
		consumedMu.Unlock()
		return false, 0, errors.New("too many attempts for this challenge")
	}

	return false, 0, errors.New("incorrect code. Watch the motion carefully or replay")
}

func init() {
	ConfigureSecret("")
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			now := time.Now().UnixMilli()
			activeMu.Lock()
			for id, rec := range activeTokens {
				if now > rec.ExpiresAt {
					delete(activeTokens, id)
				}
			}
			activeMu.Unlock()

			consumedMu.Lock()
			if len(consumedTokens) > 10000 {
				consumedTokens = make(map[string]bool)
			}
			consumedMu.Unlock()
		}
	}()
}

// ConfigureSecret sets the HMAC signing key from env (32+ chars recommended). Empty uses a dev-only default.
func ConfigureSecret(raw string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "gotcha_dev_hmac_change_me_before_production"
	}
	activeMu.Lock()
	secretKey = []byte(raw)
	activeMu.Unlock()
}

func configuredSecret() []byte {
	activeMu.RLock()
	defer activeMu.RUnlock()
	return append([]byte(nil), secretKey...)
}
