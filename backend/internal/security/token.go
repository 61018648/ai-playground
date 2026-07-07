package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Claims struct {
	UserID string `json:"sub"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Exp    int64  `json:"exp"`
}

func SignToken(secret string, claims Claims, ttl time.Duration) (string, error) {
	claims.Exp = time.Now().Add(ttl).Unix()
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payloadPart := base64.RawURLEncoding.EncodeToString(payload)
	signature := sign(secret, payloadPart)
	return payloadPart + "." + signature, nil
}

func VerifyToken(secret, token string) (Claims, error) {
	var claims Claims
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return claims, errors.New("invalid token")
	}
	expected := sign(secret, parts[0])
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return claims, errors.New("invalid signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return claims, err
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return claims, err
	}
	if claims.Exp < time.Now().Unix() {
		return claims, errors.New("token expired")
	}
	if claims.UserID == "" {
		return claims, errors.New("missing user id")
	}
	return claims, nil
}

func sign(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func BearerToken(header string) (string, error) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", fmt.Errorf("missing bearer token")
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix)), nil
}
