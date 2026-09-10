package auth

import "testing"

func TestPasswordRoundTrip(t *testing.T) {
	h, err := HashPassword("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword(h, "correct horse") {
		t.Fatal("expected match")
	}
	if VerifyPassword(h, "wrong") {
		t.Fatal("expected mismatch")
	}
	if VerifyPassword("garbage", "x") {
		t.Fatal("garbage hash should not verify")
	}
}

func TestAccessToken(t *testing.T) {
	tok, err := SignAccessToken("s", "user123", 60e9)
	if err != nil {
		t.Fatal(err)
	}
	id, err := ParseAccessToken("s", tok)
	if err != nil || id != "user123" {
		t.Fatalf("got %q %v", id, err)
	}
	if _, err := ParseAccessToken("other", tok); err == nil {
		t.Fatal("wrong secret should fail")
	}
}
