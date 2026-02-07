package aaa

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const secretKey = "something secret here" // token sign key
const adminRole = "superuser"             // token subject

// Authentication, Authorization, Accounting
type AAA struct {
	users    map[string]string
	tokenTTL time.Duration
	log      *slog.Logger
}

func New(tokenTTL time.Duration, log *slog.Logger) (AAA, error) {
	const adminUser = "ADMIN_USER"
	const adminPass = "ADMIN_PASSWORD"
	user, ok := os.LookupEnv(adminUser)
	if !ok {
		return AAA{}, fmt.Errorf("could not get admin user from enviroment")
	}
	password, ok := os.LookupEnv(adminPass)
	if !ok {
		return AAA{}, fmt.Errorf("could not get admin password from enviroment")
	}

	return AAA{
		users:    map[string]string{user: password},
		tokenTTL: tokenTTL,
		log:      log,
	}, nil
}

func (a AAA) Login(name, password string) (string, error) {

	expectedPass, ok := a.users[name] // по ключу name проверяем значение пароля
	if !ok || expectedPass != password {
		return "", errors.New("invalid credentials")
	}

	claims := jwt.RegisteredClaims{
		Subject:   adminRole,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.tokenTTL)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	// header/payload
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256, // {alg: HS256, type: jwt} header
		claims)                 // payload

	// signature - подписываем токен
	signedToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		a.log.Error("cannot sign token", "error", err)
	}

	// header.payload.signature
	return signedToken, nil
}

func (a AAA) Verify(tokenString string) error {

	if tokenString == "" {
		return errors.New("empty token")
	}
	/*
		&jwt.RegisteredClaims{} - "контейнер", куда библиотека будет распаковывать payload токена
		func(t *jwt.Token) - проверка подписи токена
	*/

	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		return err
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return errors.New("invalid token")
	}

	if claims.Subject != adminRole {
		return errors.New("forbidden")
	}

	return nil
}
