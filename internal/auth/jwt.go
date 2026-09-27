package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

type JWT struct{ secret []byte }

type jwtClaims struct {
	Subject   string `json:"sub"`
	SessionID string `json:"jti"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

func NewJWT(secret string) *JWT { return &JWT{secret: []byte(secret)} }

func (j *JWT) Sign(userID, sessionID string, expiresAt time.Time) (string, error) {
	header, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	claims, err := json.Marshal(jwtClaims{Subject: userID, SessionID: sessionID, IssuedAt: time.Now().UTC().Unix(), ExpiresAt: expiresAt.Unix()})
	if err != nil {
		return "", err
	}
	unsigned := encodeJWTPart(header) + "." + encodeJWTPart(claims)
	mac := hmac.New(sha256.New, j.secret)
	_, _ = mac.Write([]byte(unsigned))
	return unsigned + "." + encodeJWTPart(mac.Sum(nil)), nil
}

func (j *JWT) Verify(token string) (jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return jwtClaims{}, ErrInvalidToken
	}
	var header struct {
		Algorithm string `json:"alg"`
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || json.Unmarshal(headerJSON, &header) != nil || header.Algorithm != "HS256" {
		return jwtClaims{}, ErrInvalidToken
	}
	mac := hmac.New(sha256.New, j.secret)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(signature, mac.Sum(nil)) {
		return jwtClaims{}, ErrInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return jwtClaims{}, ErrInvalidToken
	}
	var claims jwtClaims
	if json.Unmarshal(payload, &claims) != nil || claims.Subject == "" || claims.SessionID == "" || claims.ExpiresAt <= time.Now().UTC().Unix() {
		return jwtClaims{}, ErrInvalidToken
	}
	return claims, nil
}

func encodeJWTPart(value []byte) string { return base64.RawURLEncoding.EncodeToString(value) }
