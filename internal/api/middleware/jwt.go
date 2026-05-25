package middleware

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

// jwtParser JWT解析器
type jwtParser struct {
	secret string
}

// Parse 解析JWT令牌
func (p *jwtParser) Parse(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("无效的签名方法")
		}
		return []byte(p.secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("无效的令牌")
	}

	return claims, nil
}
