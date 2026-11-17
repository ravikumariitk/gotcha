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

## API

- `ConfigureSecret(string)` configures the signing secret.
- `NewChallenge(int, string, Appearance) *Challenge` creates a challenge.
- `RenderMotion(string, string, Appearance) *MotionCaptcha` renders media.
- `VerifyCode(string, string) (bool, int64, error)` verifies an answer.
- `ParseAppearance([]string, string) Appearance` validates presentation options.

Run the checks locally with:

```bash
go test ./...
go vet ./...
```

## License

MIT
