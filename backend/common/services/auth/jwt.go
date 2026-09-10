package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

// SignToken signs claims with HS256. Portals build their own claims type
// (UserClaims / AdminClaims) carrying jwt.RegisteredClaims.
func SignToken(secret string, claims jwt.Claims) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}
