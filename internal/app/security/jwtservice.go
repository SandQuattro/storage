package jwtservice

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/golang-jwt/jwt"
	"github.com/gurkankaymak/hocon"
	"os"
	"strings"
	"time"
)

func ValidateToken(tokenStr string, config *hocon.Config) (bool, error) {
	publicKey := ReadPublicPEMKey()

	// проверка токена
	tok, err := jwt.Parse(strings.ReplaceAll(tokenStr, "Bearer ", ""), func(jwtToken *jwt.Token) (interface{}, error) {
		if _, ok := jwtToken.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected method: %s", jwtToken.Header["alg"])
		}
		return publicKey, nil
	})

	if err != nil {
		return false, err
	}

	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok || !tok.Valid {
		return false, fmt.Errorf("invalid token, claims parse error: %w", err)
	}

	if !claims.VerifyExpiresAt(time.Now().Unix(), true) {
		return false, fmt.Errorf("token expired")
	}

	if !claims.VerifyIssuer(config.GetString("jwt.issuer"), true) {
		return false, fmt.Errorf("token issuer error")
	}

	if !claims.VerifyAudience(config.GetString("jwt.audience"), true) {
		return false, fmt.Errorf("token audience error")
	}

	return true, nil
}

func ReadPublicPEMKey() *rsa.PublicKey {
	// Читаем открытый ключ
	keyBytes, err := os.ReadFile("conf/keys/public.pem")
	if err != nil {
		panic(err)
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		panic(err)
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil
	}

	switch t := publicKey.(type) {
	case *rsa.PublicKey:
		return t
	default:
		panic("unknown type of public key")
	}
}
