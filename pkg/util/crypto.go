// Package util 提供通用工具函数：哈希、随机 token、JWT 等。
package util

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SHA1 计算数据的 SHA1 摘要（BitTorrent info hash 即用此算法）。
func SHA1(data []byte) [20]byte {
	h := sha1.Sum(data)
	return h
}

// SHA1Hex 返回 SHA1 的十六进制字符串（用于展示与数据库存储）。
func SHA1Hex(data []byte) string {
	return hex.EncodeToString(SHA1(data))
}

// RandToken 生成 n 字节的随机 token（十六进制字符串）。
// 用于 passkey、邀请码等场景。
func RandToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// JWT claims 结构。userID 与 isAdmin 注入 token，供 web 中间件解析。
type Claims struct {
	UserID  int64 `json:"uid"`
	IsAdmin bool  `json:"adm"`
	jwt.RegisteredClaims
}

// SignToken 使用 secret 签发 JWT，有效期由 ttl 指定。
func SignToken(secret string, userID int64, isAdmin bool, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID:  userID,
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseToken 校验 JWT 并返回 claims。
func ParseToken(secret, tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}
