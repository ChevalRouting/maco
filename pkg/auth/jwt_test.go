package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/m-vinc/maco/pkg/types"
)

func TestTokenValidation(t *testing.T) {
	secret := []byte("test-secret")
	user := types.User{ID: "user-1", Username: "admin", PasswordHash: "hash"}
	token, err := IssueUserToken(secret, user, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	claims, err := ParseClaims(secret, token)
	if err != nil || claims.UserID != user.ID || claims.Credentials != CredentialVersion(user) {
		t.Fatalf("claims: %+v %v", claims, err)
	}

	for _, bad := range []string{"", strings.Repeat(".", MaxTokenBytes+1), token + "."} {
		if _, err := ParseClaims(secret, bad); err == nil {
			t.Fatal("accepted malformed token")
		}
	}

	for _, tc := range []struct {
		method jwt.SigningMethod
		claims jwt.RegisteredClaims
	}{
		{jwt.SigningMethodHS384, claims.RegisteredClaims},
		{jwt.SigningMethodHS256, jwt.RegisteredClaims{Subject: "admin", IssuedAt: jwt.NewNumericDate(time.Now())}},
		{jwt.SigningMethodHS256, jwt.RegisteredClaims{Subject: "admin", IssuedAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Second))}},
	} {
		bad, err := jwt.NewWithClaims(tc.method, tc.claims).SignedString(secret)
		if err != nil {
			t.Fatal(err)
		}

		if _, err := ParseClaims(secret, bad); err == nil {
			t.Fatal("accepted invalid claims or algorithm")
		}
	}
}
