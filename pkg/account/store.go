package account

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Stats struct{ TotalChallenges, Passed, Blocked int }
type Key struct {
	ID, Value, Name string
	CreatedAt       time.Time
	Stats           Stats
}
type User struct {
	ID, Email, Name string
	passwordHash    []byte
	Keys            []*Key
}

var mu sync.RWMutex
var users = map[string]*User{}
var sessions = map[string]string{}
var keyIndex = map[string]*Key{}

const LiveTestKey = "gotcha_live_7dfac878b4f2697edb4d9b98a5aca844d67c"

func init() {
	keyIndex[LiveTestKey] = &Key{ID: "key_live_test", Value: LiveTestKey, Name: "Production", CreatedAt: time.Now()}
}

func attachLiveKeyLocked(u *User) {
	if u == nil {
		return
	}
	for _, existing := range u.Keys {
		if existing.Value == LiveTestKey {
			return
		}
	}
	if k := keyIndex[LiveTestKey]; k != nil {
		u.Keys = append(u.Keys, k)
	}
}

func AttachLiveKey(u *User) {
	if u == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	attachLiveKeyLocked(u)
}

func random(prefix string) string {
	b := make([]byte, 18)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}
func Register(name, email, password string) (*User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if name == "" || !strings.Contains(email, "@") || len(password) < 8 {
		return nil, "", errors.New("name, valid email, and an 8-character password are required")
	}
	mu.Lock()
	defer mu.Unlock()
	if users[email] != nil {
		return nil, "", errors.New("an account already exists for this email")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}
	u := &User{ID: random("usr_"), Name: name, Email: email, passwordHash: hash}
	attachLiveKeyLocked(u)
	users[email] = u
	s := random("ses_")
	sessions[s] = email
	return u, s, nil
}
func Login(email, password string) (*User, string, error) {
	mu.Lock()
	defer mu.Unlock()
	u := users[strings.ToLower(strings.TrimSpace(email))]
	if u == nil || bcrypt.CompareHashAndPassword(u.passwordHash, []byte(password)) != nil {
		return nil, "", errors.New("invalid email or password")
	}
	s := random("ses_")
	sessions[s] = u.Email
	attachLiveKeyLocked(u)
	return u, s, nil
}
func FromSession(token string) *User { mu.RLock(); defer mu.RUnlock(); return users[sessions[token]] }
func CreateKey(u *User, name string) *Key {
	mu.Lock()
	defer mu.Unlock()
	if name == "" {
		name = "Production"
	}
	k := &Key{ID: random("key_"), Value: random("gotcha_live_"), Name: name, CreatedAt: time.Now()}
	u.Keys = append(u.Keys, k)
	keyIndex[k.Value] = k
	return k
}
func KeyFor(value string) *Key { mu.RLock(); defer mu.RUnlock(); return keyIndex[value] }
func Record(key *Key, outcome string) {
	if key == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if outcome == "challenge" {
		key.Stats.TotalChallenges++
	}
	if outcome == "passed" {
		key.Stats.Passed++
	}
	if outcome == "blocked" {
		key.Stats.Blocked++
	}
}
