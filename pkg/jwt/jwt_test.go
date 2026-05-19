package jwt

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndParseToken(t *testing.T) {
	secret := []byte("test-secret-key-for-unit-test")

	claims := Claims{
		Uuid:    "U2024010112345",
		IsAdmin: 0,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    Issuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("生成token失败: %v", err)
	}

	parsed, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(Issuer),
	)
	if err != nil {
		t.Fatalf("解析token失败: %v", err)
	}

	parsedClaims, ok := parsed.Claims.(*Claims)
	if !ok {
		t.Fatal("claims类型断言失败")
	}
	if parsedClaims.Uuid != "U2024010112345" {
		t.Errorf("uuid不匹配: got %s, want U2024010112345", parsedClaims.Uuid)
	}
	if parsedClaims.IsAdmin != 0 {
		t.Errorf("isAdmin不匹配: got %d, want 0", parsedClaims.IsAdmin)
	}
	t.Log("✅ 正常生成和解析token: 通过")
}

func TestExpiredToken(t *testing.T) {
	secret := []byte("test-secret-key-for-unit-test")

	claims := Claims{
		Uuid:    "U2024010112345",
		IsAdmin: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    Issuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(secret)

	_, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(Issuer),
	)

	if err == nil {
		t.Fatal("过期token应该解析失败")
	}
	if !errors.Is(err, jwt.ErrTokenExpired) {
		t.Errorf("应该是 ErrTokenExpired 错误, got: %v", err)
	}
	t.Log("✅ 过期token被拒绝: 通过")
}

func TestMissingExpiration(t *testing.T) {
	secret := []byte("test-secret-key-for-unit-test")

	claims := Claims{
		Uuid:    "U2024010112345",
		IsAdmin: 0,
		RegisteredClaims: jwt.RegisteredClaims{
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    Issuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(secret)

	_, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(Issuer),
	)

	if err == nil {
		t.Fatal("无exp的token应该被拒绝")
	}
	t.Logf("✅ 无exp token被拒绝: 通过 (err: %v)", err)
}

func TestWrongIssuer(t *testing.T) {
	secret := []byte("test-secret-key-for-unit-test")

	claims := Claims{
		Uuid:    "U2024010112345",
		IsAdmin: 0,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "fake-issuer",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(secret)

	_, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(Issuer),
	)

	if err == nil {
		t.Fatal("错误issuer的token应该被拒绝")
	}
	if !errors.Is(err, jwt.ErrTokenInvalidIssuer) {
		t.Errorf("应该是 ErrTokenInvalidIssuer 错误, got: %v", err)
	}
	t.Log("✅ 错误issuer token被拒绝: 通过")
}

func TestWrongSigningMethod(t *testing.T) {
	secret := []byte("test-secret-key-for-unit-test")

	claims := Claims{
		Uuid:    "U2024010112345",
		IsAdmin: 0,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    Issuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
	tokenString, _ := token.SignedString(secret)

	_, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(Issuer),
	)

	if err == nil {
		t.Fatal("HS384签名的token应该被拒绝")
	}
	t.Log("✅ 非HS256签名token被拒绝: 通过")
}

func TestNotBeforeFuture(t *testing.T) {
	secret := []byte("test-secret-key-for-unit-test")

	claims := Claims{
		Uuid:    "U2024010112345",
		IsAdmin: 0,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    Issuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(secret)

	_, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(Issuer),
	)

	if err == nil {
		t.Fatal("nbf在未来的token应该被拒绝")
	}
	if !errors.Is(err, jwt.ErrTokenNotValidYet) {
		t.Errorf("应该是 ErrTokenNotValidYet 错误, got: %v", err)
	}
	t.Log("✅ nbf在未来的token被拒绝: 通过")
}

func TestErrorTypeDifferentiation(t *testing.T) {
	secret := []byte("test-secret-key-for-unit-test")

	// 测试过期token的错误类型可以被 errors.Is 识别
	claims := Claims{
		Uuid:    "U2024010112345",
		IsAdmin: 0,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    Issuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(secret)

	_, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(Issuer),
	)

	if err == nil {
		t.Fatal("应该有错误")
	}

	// 模拟 middleware 中的 errors.Is 判断
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		t.Log("✅ 过期错误被正确识别为 ErrTokenExpired: 通过")
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		t.Error("不应被识别为 ErrTokenNotValidYet")
	case errors.Is(err, jwt.ErrTokenMalformed):
		t.Error("不应被识别为 ErrTokenMalformed")
	default:
		t.Errorf("未知的错误类型: %v", err)
	}
}
