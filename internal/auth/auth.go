package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"roster_dashboard_go/internal/app"
)

type Claims struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Role     string `json:"role"`
	TeamID   *int64 `json:"team_id"`
	Exp      int64  `json:"exp"`
}

func FromUser(user app.User, ttl time.Duration) Claims {
	return Claims{
		ID: user.ID, Name: user.Name, Username: user.Username, Role: user.Role,
		TeamID: user.TeamID, Exp: time.Now().Add(ttl).Unix(),
	}
}

func (c Claims) User() app.User {
	return app.User{ID: c.ID, Name: c.Name, Username: c.Username, Role: c.Role, TeamID: c.TeamID}
}

func Sign(claims Claims, secret string) (string, error) {
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	h := base64.RawURLEncoding.EncodeToString(header)
	p := base64.RawURLEncoding.EncodeToString(payload)
	input := h + "." + p
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(input))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return input + "." + sig, nil
}

func Verify(token, secret string) (Claims, error) {
	var claims Claims
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return claims, errors.New("invalid token")
	}
	input := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(input))
	expected := mac.Sum(nil)
	actual, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(expected, actual) {
		return claims, errors.New("invalid token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, err
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return claims, err
	}
	if claims.Exp == 0 || time.Now().Unix() > claims.Exp {
		return claims, errors.New("expired token")
	}
	return claims, nil
}

func SecretFromEnv(sessionSecret, jwtSecret, env string) (string, error) {
	secret := sessionSecret
	if secret == "" {
		secret = jwtSecret
	}
	if secret == "" {
		secret = "roster-secret-key-change-in-prod"
	}
	if strings.EqualFold(env, "production") && secret == "roster-secret-key-change-in-prod" {
		return "", fmt.Errorf("SESSION_SECRET or JWT_SECRET must be set in production")
	}
	return secret, nil
}

func Int64Param(v string) (int64, error) {
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}
