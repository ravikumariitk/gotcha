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
