package jwt

import (
	"errors"
	"gochat/internal/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const Issuer = "gochat"

// Claims JWT 载荷
type Claims struct {
	Uuid    string `json:"uuid"`
	IsAdmin int8   `json:"isAdmin"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT token
func GenerateToken(uuid string, isAdmin int8) (string, error) {
	conf := config.GetConfig()
	expireHours := conf.JwtConfig.ExpireHours
	if expireHours <= 0 {
		expireHours = 24
	}

	now := time.Now()
	claims := Claims{
		Uuid:    uuid,
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expireHours) * time.Hour)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    Issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(conf.JwtConfig.Secret))
}

// ParseToken 解析 JWT token
func ParseToken(tokenString string) (*Claims, error) {
	conf := config.GetConfig()
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(conf.JwtConfig.Secret), nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(Issuer),
	)

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
