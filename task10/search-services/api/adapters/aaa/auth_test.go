package aaa

import (
	"os"
	"testing"
	"time"

	"io"
	"log/slog"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNew_Success(t *testing.T) {
	t.Setenv("ADMIN_USER", "admin")
	t.Setenv("ADMIN_PASSWORD", "secret")

	tokenTTL := 2 * time.Minute

	a, err := New(tokenTTL, newTestLogger())
	require.NoError(t, err)

	require.Len(t, a.users, 1)
	assert.Equal(t, "secret", a.users["admin"])
	assert.Equal(t, tokenTTL, a.tokenTTL)
}

func TestNew_MissingAdminUser(t *testing.T) {
	// гарантируем отсутствие переменных
	_ = os.Unsetenv("ADMIN_USER")
	_ = os.Unsetenv("ADMIN_PASSWORD")

	_, err := New(time.Minute, newTestLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "admin user")
}

func TestNew_MissingAdminPassword(t *testing.T) {
	t.Setenv("ADMIN_USER", "admin")
	_ = os.Unsetenv("ADMIN_PASSWORD")

	_, err := New(time.Minute, newTestLogger())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "admin password")
}

func TestAAA_Login_InvalidCredentials(t *testing.T) {
	a := AAA{
		users:    map[string]string{"admin": "password"},
		tokenTTL: time.Minute,
		log:      newTestLogger(),
	}

	testCases := []struct {
		name     string
		user     string
		password string
	}{
		{"wrong password", "admin", "wrong"},
		{"unknown user", "other", "password"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, err := a.Login(tc.user, tc.password)
			require.Error(t, err)
			assert.Equal(t, "invalid credentials", err.Error())
			assert.Empty(t, token)
		})
	}
}

func TestAAA_Verify_ValidToken(t *testing.T) {
	a := AAA{
		users:    nil,
		tokenTTL: time.Minute,
		log:      newTestLogger(),
	}

	// генерируем валидный токен с нужным subject
	claims := jwt.RegisteredClaims{
		Subject:   adminRole,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	tk := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tk.SignedString([]byte(secretKey))
	require.NoError(t, err)

	err = a.Verify(signed)
	require.NoError(t, err)
}
