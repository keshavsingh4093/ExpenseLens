package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"smartreceipt/backend/common/pkg/middleware"
)

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

type testClaims struct {
	UserID string `json:"userId"`
	jwt.RegisteredClaims
}

func TestSignAndParseToken(t *testing.T) {
	middleware.ConfigureAuth("s", func() jwt.Claims { return &testClaims{} })
	now := time.Now()
	tok, err := SignToken("s", &testClaims{UserID: "user123", RegisteredClaims: jwt.RegisteredClaims{
		IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
	}})
	if err != nil {
		t.Fatal(err)
	}
	c, err := middleware.ParseToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	if c.(*testClaims).UserID != "user123" {
		t.Fatalf("got %+v", c)
	}
	middleware.ConfigureAuth("other", func() jwt.Claims { return &testClaims{} })
	if _, err := middleware.ParseToken(tok); err == nil {
		t.Fatal("wrong secret should fail")
	}
	middleware.ConfigureAuth("s", func() jwt.Claims { return &testClaims{} })
	expired, _ := SignToken("s", &testClaims{RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(now.Add(-time.Minute))}})
	if _, err := middleware.ParseToken(expired); err == nil {
		t.Fatal("expired token should fail")
	}
	noExp, _ := SignToken("s", &testClaims{UserID: "x"})
	if _, err := middleware.ParseToken(noExp); err == nil {
		t.Fatal("token without exp should fail")
	}
}
