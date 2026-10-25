package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func NewToken(userID, secret string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func ParseToken(token, secret string) (string, error) {
	claims := &jwt.RegisteredClaims{}
	t, err := jwt.ParseWithClaims(token, claims,
		func(t *jwt.Token) (any, error) { return []byte(secret), nil },
		jwt.WithValidMethods([]string{"HS256"}), // rejects other algorithms
	)
	if err != nil || !t.Valid || claims.Subject == "" {
		return "", errors.New("invalid token")
	}
	return claims.Subject, nil
}