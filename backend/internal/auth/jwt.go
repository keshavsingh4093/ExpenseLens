package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims carried by the access token.
type Claims struct {
	jwt.RegisteredClaims
}

// SignAccessToken issues a short-lived HS256 token for userID.
func SignAccessToken(secret string, userID string, ttl time.Duration) (string, error) {
	now := time.Now()
	c := Claims{RegisteredClaims: jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		Issuer:    "expenselens",
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(secret))
}

// ParseAccessToken validates the token and returns the user id.
func ParseAccessToken(secret, token string) (string, error) {
	var c Claims
	t, err := jwt.ParseWithClaims(token, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	}, jwt.WithIssuer("expenselens"), jwt.WithExpirationRequired())
	if err != nil || !t.Valid || c.Subject == "" {
		return "", errors.New("invalid token")
	}
	return c.Subject, nil
}
