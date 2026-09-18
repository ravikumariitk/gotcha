package main

import (
	"fmt"
	"testing"

	"checkhuman/pkg/engine"
)

func TestEngine(t *testing.T) {
	fmt.Println("🧪 Testing Go GOTCHA Engine...")

	// 1. Random code
	code := engine.GenerateRandomCode(5)
	if len(code) != 5 {
		t.Fatalf("Expected 5 digits, got %d", len(code))
	}
	fmt.Println("✅ Random code generated:", code)

	// 2. Challenge issue and verification
	cid, token, _ := engine.IssueChallenge("95821")
	if cid == "" || token == "" {
		t.Fatalf("Failed to issue challenge")
	}

	// 2a. Wrong answer fails
	valid, _, err := engine.VerifyAnswer(token, "00000")
	if valid {
		t.Fatalf("Expected wrong answer to fail, but succeeded")
	}
	fmt.Println("✅ Wrong answer rejected properly:", err)

	// 2b. Correct answer succeeds
	valid, lat, err := engine.VerifyAnswer(token, "95821")
	if !valid || err != nil {
		t.Fatalf("Expected correct answer to pass, got err: %v", err)
	}
	fmt.Printf("✅ Correct answer passed with latency %dms\n", lat)

	// 2c. Anti-replay fails on second try
	valid, _, err = engine.VerifyAnswer(token, "95821")
	if valid {
		t.Fatalf("Expected replay attack to fail, but passed")
	}
	fmt.Println("✅ Anti-replay verified:", err)

	// 3. Motion frames generation (GOTCHA version 1 style MP4)
	motion := engine.GenerateMotionFrames("12345", "standard", engine.ParseAppearance(nil, "noise"))
	if motion.Video == "" && len(motion.Frames) == 0 {
		t.Fatalf("Expected video or frames from motion generator")
	}
	if motion.Width != 960 || motion.Height != 400 {
		t.Fatalf("Expected 960x400, got %dx%d", motion.Width, motion.Height)
	}
	fmt.Printf("✅ Motion generator synthesized %d frames (video=%t)\n", motion.FrameCount, motion.Video != "")
}
