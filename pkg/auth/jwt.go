package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/m-vinc/maco/pkg/types"
)

const MaxTokenBytes = 4096

const (
	RoleAdmin  = "admin"
	RoleViewer = "viewer"
)

func ValidRole(role string) bool {
	return role == RoleAdmin || role == RoleViewer
}

func CanMutate(role string) bool {
	return role == RoleAdmin
}

type Claims struct {
	jwt.RegisteredClaims
	UserID      string `json:"uid"`
	Credentials string `json:"credentials"`
	Role        string `json:"role"`
}

func CredentialVersion(user types.User) string {
	sum := sha256.Sum256([]byte(user.ID + "\x00" + user.PasswordHash))
	return hex.EncodeToString(sum[:])
}

func IssueUserToken(secret []byte, user types.User, ttl time.Duration) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: user.Username, IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl))},
		UserID:           user.ID, Credentials: CredentialVersion(user), Role: user.Role,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func ParseClaims(secret []byte, token string) (*Claims, error) {
	if len(token) == 0 || len(token) > MaxTokenBytes {
		return nil, fmt.Errorf("invalid token length")
	}

	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		return secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil {
		return nil, err
	}

	if !parsed.Valid || claims.Subject == "" || claims.IssuedAt == nil {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
