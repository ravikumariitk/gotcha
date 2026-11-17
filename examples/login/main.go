package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	gotcha "github.com/ravikumariitk/gotcha"
)

func main() {
	secret := os.Getenv("GOTCHA_SECRET")
	if secret == "" {
		log.Fatal("GOTCHA_SECRET must be set")
	}
	gotcha.ConfigureSecret(secret)

	mux := http.NewServeMux()
	mux.HandleFunc("/", loginPage)
	mux.HandleFunc("/challenge", issueChallenge)
	mux.HandleFunc("/verify", verifyChallenge)

	log.Println("login demo running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func loginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(loginHTML))
}

func issueChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	challenge := gotcha.NewChallenge(
		5,
		"standard",
		gotcha.ParseAppearance([]string{"#0ea5e9", "#f8fafc"}, "noise"),
	)
	writeJSON(w, http.StatusOK, challenge)
}

func verifyChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	var request struct {
		Token  string `json:"token"`
		Answer string `json:"answer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "invalid request",
		})
		return
	}

	valid, latency, err := gotcha.VerifyCode(request.Token, request.Answer)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": valid,
		"latency": latency,
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

const loginHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Gotcha login demo</title>
  <style>
    body { font: 16px system-ui, sans-serif; max-width: 34rem; margin: 3rem auto; padding: 0 1rem; }
    video, img { display: block; width: 100%; max-width: 480px; background: #111827; border-radius: .5rem; }
    input, button { font: inherit; padding: .65rem; margin-top: .75rem; }
    input { width: 10rem; } button { cursor: pointer; }
  </style>
</head>
<body>
  <h1>Sign in</h1>
  <p>Type the number shown in the moving challenge.</p>
  <video id="challenge-video" autoplay muted loop playsinline hidden></video>
  <img id="challenge-frame" alt="Human verification challenge" hidden>
  <form id="login-form">
    <input id="answer" inputmode="numeric" autocomplete="off" aria-label="Challenge answer" required>
    <button type="submit">Continue</button>
  </form>
  <p id="status" role="status"></p>
  <script>
    let token = "";
    let frameTimer;

    async function loadChallenge() {
      const response = await fetch('/challenge');
      if (!response.ok) throw new Error('Could not load challenge');
      const challenge = await response.json();
      token = challenge.token;

      const motion = challenge.motion;
      const video = document.getElementById('challenge-video');
      const frame = document.getElementById('challenge-frame');
      video.poster = 'data:image/png;base64,' + motion.poster;

      if (motion.video && motion.mime) {
        video.src = 'data:' + motion.mime + ';base64,' + motion.video;
        video.hidden = false;
        frame.hidden = true;
        return;
      }

      const frames = motion.frames || [];
      let index = 0;
      frame.hidden = false;
      video.hidden = true;
      const showFrame = () => {
        if (frames.length === 0) return;
        frame.src = 'data:image/png;base64,' + frames[index];
        index = (index + 1) % frames.length;
      };
      showFrame();
      clearInterval(frameTimer);
      frameTimer = setInterval(showFrame, 1000 / (motion.fps || 30));
    }

    document.getElementById('login-form').addEventListener('submit', async (event) => {
      event.preventDefault();
      const status = document.getElementById('status');
      const response = await fetch('/verify', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({token, answer: document.getElementById('answer').value})
      });
      const result = await response.json();
      status.textContent = result.success ? 'Challenge passed.' : (result.error || 'Try again.');
      if (result.success) document.getElementById('answer').disabled = true;
    });

    loadChallenge().catch(error => { document.getElementById('status').textContent = error.message; });
  </script>
</body>
</html>`
