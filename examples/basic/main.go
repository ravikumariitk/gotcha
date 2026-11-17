package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ravikumariitk/gotcha"
)

func main() {
	gotcha.ConfigureSecret("replace-with-a-secure-secret")

	http.HandleFunc("/challenge", func(w http.ResponseWriter, r *http.Request) {
		challenge := gotcha.NewChallenge(5, "standard", gotcha.ParseAppearance([]string{"#0ea5e9", "#f8fafc"}, "noise"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":        challenge.ID,
			"token":     challenge.Token,
			"expiresAt": challenge.ExpiresAt,
			"mime":      challenge.Motion.Mime,
			"video":     challenge.Motion.Video,
			"poster":    challenge.Motion.Poster,
			"width":     challenge.Motion.Width,
			"height":    challenge.Motion.Height,
		})
	})

	http.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Token  string `json:"token"`
			Answer string `json:"answer"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		valid, latency, err := gotcha.VerifyCode(req.Token, req.Answer)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": valid,
			"latency": latency,
		})
	})

	fmt.Println("server running on http://localhost:8080")
	_ = http.ListenAndServe(":8080", nil)
}
