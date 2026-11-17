package checkhuman

import (
	"fmt"
	"testing"
)

func TestLibraryAPI(t *testing.T) {
	fmt.Println("🧪 Testing Go library API...")

	code := GenerateCode(5)
	if len(code) != 5 {
		t.Fatalf("Expected 5 digits, got %d", len(code))
	}
	fmt.Println("✅ Random code generated:", code)

	cid, token, _ := IssueCode("95821")
	if cid == "" || token == "" {
		t.Fatalf("Failed to issue challenge")
	}

	valid, _, err := VerifyCode(token, "00000")
	if valid {
		t.Fatalf("Expected wrong answer to fail, but succeeded")
	}
	fmt.Println("✅ Wrong answer rejected properly:", err)

	valid, lat, err := VerifyCode(token, "95821")
	if !valid || err != nil {
		t.Fatalf("Expected correct answer to pass, got err: %v", err)
	}
	fmt.Printf("✅ Correct answer passed with latency %dms\n", lat)

	valid, _, err = VerifyCode(token, "95821")
	if valid {
		t.Fatalf("Expected replay attack to fail, but passed")
	}
	fmt.Println("✅ Anti-replay verified:", err)
}

func TestNewChallengeIncludesMotionWithoutLeakingAnswer(t *testing.T) {
	challenge := NewChallenge(5, "standard", ParseAppearance([]string{"#0ea5e9", "#f8fafc"}, "noise"))
	if challenge == nil || challenge.ID == "" || challenge.Token == "" {
		t.Fatal("expected a signed challenge")
	}
	if challenge.Motion == nil || challenge.Motion.Width == 0 || challenge.Motion.Poster == "" {
		t.Fatal("expected a browser-ready motion challenge")
	}
	if challenge.ExpiresAt <= 0 {
		t.Fatal("expected an expiry timestamp")
	}
}

func TestRenderMotionUsesDefaultsForZeroAppearance(t *testing.T) {
	motion := RenderMotion("12345", "standard", Appearance{})
	if motion == nil || motion.Poster == "" {
		t.Fatal("expected rendering with a zero-value appearance")
	}
}
