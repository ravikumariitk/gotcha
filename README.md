# gotcha

`gotcha` is a small Go library for human-readable motion CAPTCHA challenges.
It creates signed, expiring challenge tokens, renders browser-ready media, and
rejects incorrect, expired, or replayed answers.

## Install

```bash
go get github.com/ravikumariitk/gotcha
```

## Usage

```go
package main

import gotcha "github.com/ravikumariitk/gotcha"

func main() {
	gotcha.ConfigureSecret("replace-with-a-long-random-secret")

	challenge := gotcha.NewChallenge(
		5,
		"standard",
		gotcha.ParseAppearance([]string{"#0ea5e9", "#f8fafc"}, "noise"),
	)

	// Return challenge.Token and challenge.Motion to your frontend.
	// Read the user's answer from your form or API request.
	userAnswer := "12345"
	valid, latency, err := gotcha.VerifyCode(challenge.Token, userAnswer)
	_, _, _ = valid, latency, err
}
```

`NewChallenge` never returns the answer code. The frontend displays the motion
media and submits the signed token with the user's answer to your backend.

## Full login flow

The repository includes a runnable login flow in `examples/login`. It serves a
page, creates a challenge, keeps the token in browser memory, renders either
MP4 or animated PNG frames, and submits the token with the user's answer.

Run it with a production-style secret:

```bash
GOTCHA_SECRET="$(openssl rand -hex 32)" go run ./examples/login
```

Open <http://localhost:8080>. The example has three steps:

1. `GET /challenge` returns the token, expiry, poster, and motion media. The
   answer code is never returned.
2. The browser keeps `challenge.token` client-side and displays the media.
3. `POST /verify` sends `{"token":"...", "answer":"..."}` to the backend.

The demo is intentionally only the CAPTCHA portion of login. After successful
verification, connect the success branch to your own user lookup, password
check, session creation, or other authentication flow.

## Render the media in a browser

Return the challenge as JSON from your backend. The media fields are base64
encoded, so they can be used directly as browser data URLs:

```go
func challengeHandler(w http.ResponseWriter, r *http.Request) {
	challenge := gotcha.NewChallenge(5, "standard", gotcha.ParseAppearance(
		[]string{"#0ea5e9", "#f8fafc"},
		"noise",
	))

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(challenge)
}
```

For the normal MP4 response, render the poster and video like this:

```html
<video id="gotcha-video" autoplay muted loop playsinline width="480" height="160"></video>

<script>
  const challenge = await fetch('/challenge').then(response => response.json());
  const video = document.getElementById('gotcha-video');

  video.poster = `data:image/png;base64,${challenge.motion.poster}`;
  if (challenge.motion.video && challenge.motion.mime) {
    video.src = `data:${challenge.motion.mime};base64,${challenge.motion.video}`;
  }

  // Keep challenge.token in memory and send it with the user's answer.
</script>
```

When `ffmpeg` is unavailable, `challenge.motion.frames` contains base64 PNG
frames instead of a video. Display those frames in an image element and cycle
through them:

```html
<img id="gotcha-frame" width="480" height="160" alt="Human verification challenge">

<script>
  const image = document.getElementById('gotcha-frame');
  const frames = challenge.motion.frames || [];
  let frame = 0;

  function showFrame() {
    if (frames.length === 0) return;
    image.src = `data:image/png;base64,${frames[frame]}`;
    frame = (frame + 1) % frames.length;
  }

  showFrame();
  setInterval(showFrame, 1000 / (challenge.motion.fps || 30));
</script>
```

The renderer uses `ffmpeg` when available to produce MP4 output. If `ffmpeg`
is unavailable, it returns PNG frames instead, so the library remains usable
in minimal environments.

## Security and lifecycle

- Never return or expose the answer code.
- Never log the token or the user's answer.
- Keep the token in memory or in a protected form field on the client.
- Tokens expire after three minutes.
- Tokens are single-use and rejected after successful verification.
- A challenge allows at most five submissions and is invalidated after the
  fifth failed attempt.
- Always configure `GOTCHA_SECRET` in production with a long, random value.
- If no secret is configured, Gotcha uses a development-only fallback secret;
  never rely on that fallback in production.

## API

- `ConfigureSecret(string)` configures the signing secret.
- `NewChallenge(int, string, Appearance) *Challenge` creates a challenge.
- `RenderMotion(string, string, Appearance) *MotionCaptcha` renders media.
- `VerifyCode(string, string) (bool, int64, error)` verifies an answer.
- `ParseAppearance([]string, string) Appearance` validates presentation options.

## LLM setup prompt

Copy this prompt into your coding assistant to integrate Gotcha into another
Go project:

```text
Integrate github.com/ravikumariitk/gotcha into this Go project as a human-readable motion CAPTCHA.

Requirements:
1. Add the dependency with: go get github.com/ravikumariitk/gotcha
2. Configure the signing secret once at application startup using an environment variable named GOTCHA_SECRET. Never hard-code the secret and never expose it to the browser.
3. Add a GET /challenge endpoint that creates a challenge with gotcha.NewChallenge(5, "standard", gotcha.ParseAppearance([]string{"#0ea5e9", "#f8fafc"}, "noise")) and returns the challenge as JSON.
4. Add a POST /verify endpoint that accepts {"token":"...", "answer":"..."}, calls gotcha.VerifyCode, and returns a JSON success result. Return an appropriate client error for invalid, expired, replayed, or incorrect challenges.
5. Build the frontend so it renders challenge.motion.video as a base64 data URL when available, uses challenge.motion.poster as the poster, and falls back to cycling through challenge.motion.frames when video is unavailable.
6. Keep the challenge token in memory or in a protected form field. Do not display it, log it, or return the answer code.
7. Match the project's existing router, error handling, templates, API conventions, and tests. Do not introduce a new web framework.
8. Add tests covering challenge creation, successful verification, incorrect answers, expired or invalid tokens, and replay protection.
9. Run gofmt, go test ./..., go test -race ./..., and go vet ./... after the integration.

First inspect the existing project structure and explain which files you will change. Then implement the integration and show the final request/response shape for both endpoints.
```

Run the checks locally with:

```bash
go test ./...
go vet ./...
```

## License

MIT
