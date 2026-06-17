package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AdminClaims struct {
	AdminID string `json:"admin_id"`
	Email   string `json:"email"`
	Role    string `json:"role"`
	jwt.RegisteredClaims
}

func IssueAdminToken(adminID, email, role, secret string, ttl time.Duration) (string, error) {
	claims := AdminClaims{
		AdminID: adminID,
		Email:   email,
		Role:    role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			Subject:   adminID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseAdminToken(tokenStr, secret string) (*AdminClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &AdminClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*AdminClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

type MerchantClaims struct {
	UserID     string `json:"user_id"`
	MerchantID string `json:"merchant_id"`
	Email      string `json:"email"`
	jwt.RegisteredClaims
}

func IssueMerchantToken(userID, merchantID, email, secret string, ttl time.Duration) (string, error) {
	claims := MerchantClaims{
		UserID:     userID,
		MerchantID: merchantID,
		Email:      email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			Subject:   userID,
			Issuer:    "upipays-merchant",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseMerchantToken(tokenStr, secret string) (*MerchantClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &MerchantClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*MerchantClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
		if claims.Issuer != "" && claims.Issuer != "upipays-merchant" {
			return nil, errors.New("invalid token issuer")
		}
		if claims.MerchantID == "" || claims.UserID == "" {
			return nil, errors.New("invalid merchant token")
		}
		return claims, nil
}
