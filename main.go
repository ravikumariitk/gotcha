package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"checkhuman/pkg/account"
	"checkhuman/pkg/engine"
)

// Request payloads
type GenerateRequest struct {
	CustomCode   string `json:"customCode"`
	DefenseLevel string `json:"defenseLevel"`
	Digits       int    `json:"digits"`
	Appearance   struct {
		Colors  []string `json:"colors"`
		Texture string   `json:"texture"`
	} `json:"appearance"`
}

type VerifyRequest struct {
	Token  string `json:"token"`
	Answer string `json:"answer"`
}

var nonDigitRegex = regexp.MustCompile(`\D`)
var challengeKeys sync.Map // token -> *account.Key

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

// Handler: POST /api/v1/captcha/generate
func handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GenerateRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	secretCode := nonDigitRegex.ReplaceAllString(req.CustomCode, "")
	if len(secretCode) > 6 {
		secretCode = secretCode[:6]
	}
	if secretCode == "" {
		digits := req.Digits
		if digits < 4 || digits > 6 {
			digits = 5
		}
		secretCode = engine.GenerateRandomCode(digits)
	}

	defenseLevel := req.DefenseLevel
	if defenseLevel == "" {
		defenseLevel = "standard"
	}

	challengeID, token, expiresAt := engine.IssueChallenge(secretCode)
	key := account.KeyFor(r.Header.Get("X-API-Key"))
	if key != nil {
		account.Record(key, "challenge")
		challengeKeys.Store(token, key)
	}
	appearance := engine.ParseAppearance(req.Appearance.Colors, req.Appearance.Texture)
	motionCaptcha := engine.GenerateMotionFrames(secretCode, defenseLevel, appearance)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"challengeId": challengeID,
		"token":       token,
		"expiresAt":   expiresAt,
		"captcha":     motionCaptcha,
	})
}

// Handler: POST /api/v1/captcha/verify
func handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid JSON request",
		})
		return
	}

	req.Token = strings.TrimSpace(req.Token)
	req.Answer = strings.TrimSpace(req.Answer)

	valid, latencyMs, err := engine.VerifyAnswer(req.Token, req.Answer)
	if value, ok := challengeKeys.Load(req.Token); ok {
		if valid {
			account.Record(value.(*account.Key), "passed")
		} else {
			account.Record(value.(*account.Key), "blocked")
		}
		if valid {
			challengeKeys.Delete(req.Token)
		}
	}
	w.Header().Set("Content-Type", "application/json")

	if valid {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":   true,
			"verified":  true,
			"message":   "Human verified successfully! 🎉",
			"latencyMs": latencyMs,
		})
	} else {
		w.WriteHeader(http.StatusBadRequest)
		errMsg := "Verification failed"
		if err != nil {
			errMsg = err.Error()
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  false,
			"verified": false,
			"error":    errMsg,
		})
	}
}

func bearer(r *http.Request) *account.User {
	return account.FromSession(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
}
func authResponse(w http.ResponseWriter, u *account.User, token string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"user": map[string]string{"id": u.ID, "name": u.Name, "email": u.Email}, "token": token})
}
func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": message})
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&p)
	u, t, e := account.Register(p.Name, p.Email, p.Password)
	if e != nil {
		writeAuthError(w, http.StatusBadRequest, e.Error())
		return
	}
	authResponse(w, u, t)
}
func handleLogin(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&p)
	u, t, e := account.Login(p.Email, p.Password)
	if e != nil {
		writeAuthError(w, http.StatusUnauthorized, e.Error())
		return
	}
	authResponse(w, u, t)
}
func handleKeys(w http.ResponseWriter, r *http.Request) {
	u := bearer(r)
	if u == nil {
		http.Error(w, "unauthorized", 401)
		return
	}
	if r.Method == http.MethodPost {
		var p struct{ Name string }
		_ = json.NewDecoder(r.Body).Decode(&p)
		k := account.CreateKey(u, p.Name)
		json.NewEncoder(w).Encode(k)
		return
	}
	json.NewEncoder(w).Encode(u.Keys)
}
func handleDashboard(w http.ResponseWriter, r *http.Request) {
	u := bearer(r)
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	account.AttachLiveKey(u)
	total, passed, blocked := 0, 0, 0
	keys := u.Keys
	if keys == nil {
		keys = []*account.Key{}
	}
	for _, k := range keys {
		total += k.Stats.TotalChallenges
		passed += k.Stats.Passed
		blocked += k.Stats.Blocked
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"user": map[string]string{"name": u.Name, "email": u.Email}, "stats": map[string]int{"totalChallenges": total, "passed": passed, "blocked": blocked}, "keys": keys})
}

// Handler: GET /api/health
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"service": "GOTCHA Optical Human Verification (Go Backend)",
		"version": "2.0.0",
	})
}

func main() {
	mux := http.NewServeMux()

	// Locate public directory reliably
	workDir, _ := os.Getwd()
	publicDir := filepath.Join(workDir, "public")
	// Static asset handler with explicit MIME types and full buffer delivery
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reqPath := r.URL.Path
		if reqPath == "/" {
			reqPath = "/index.html"
		}
		cleanPath := filepath.Join(publicDir, filepath.Clean(reqPath))
		data, err := os.ReadFile(cleanPath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		switch filepath.Ext(cleanPath) {
		case ".html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		case ".css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case ".js":
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.Write(data)
	})

	mux.HandleFunc("/api/v1/captcha/generate", corsMiddleware(handleGenerate))
	mux.HandleFunc("/api/v1/captcha/verify", corsMiddleware(handleVerify))
	mux.HandleFunc("/api/v1/auth/register", corsMiddleware(handleRegister))
	mux.HandleFunc("/api/v1/auth/login", corsMiddleware(handleLogin))
	mux.HandleFunc("/api/v1/keys", corsMiddleware(handleKeys))
	mux.HandleFunc("/api/v1/dashboard", corsMiddleware(handleDashboard))
	mux.HandleFunc("/api/health", corsMiddleware(handleHealth))

	port := "8082"
	fmt.Printf("🚀 GOTCHA Go Service running at http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
