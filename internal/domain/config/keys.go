package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

type KeyGenerator struct{}

func NewKeyGenerator() *KeyGenerator {
	return &KeyGenerator{}
}

type RealityKeypair struct {
	PrivateKey string
	PublicKey  string
}

func (g *KeyGenerator) GenerateRealityKeypair() (*RealityKeypair, error) {
	privateKey := make([]byte, 32)
	if _, err := rand.Read(privateKey); err != nil {
		return nil, err
	}

	// 關鍵修復：使用 X25519 推導公鑰
	publicKey, err := curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("推導公鑰失敗: %w", err)
	}

	return &RealityKeypair{
		PrivateKey: base64.RawURLEncoding.EncodeToString(privateKey),
		PublicKey:  base64.RawURLEncoding.EncodeToString(publicKey),
	}, nil
}

func (g *KeyGenerator) GenerateShortID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}
