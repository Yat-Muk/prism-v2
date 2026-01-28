package acme

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-acme/lego/v4/registration"
)

type User struct {
	Email        string                 `json:"email"`
	Registration *registration.Resource `json:"registration"`
	key          crypto.PrivateKey
}

func (u *User) GetEmail() string                        { return u.Email }
func (u *User) GetRegistration() *registration.Resource { return u.Registration }
func (u *User) GetPrivateKey() crypto.PrivateKey        { return u.key }

func LoadUser(email, accountFile string) (*User, error) {
	data, err := os.ReadFile(accountFile)
	if err != nil {
		if os.IsNotExist(err) {
			return newUser(email)
		}
		return nil, fmt.Errorf("讀取賬戶文件失敗: %w", err)
	}

	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("解析賬戶數據失敗: %w", err)
	}

	keyFile := accountFile + ".key"
	keyData, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("丟失賬戶私鑰，無法恢復賬戶: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("私鑰格式錯誤")
	}
	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析私鑰失敗: %w", err)
	}
	user.key = privateKey
	return &user, nil
}

func newUser(email string) (*User, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return &User{Email: email, key: privateKey}, nil
}

func (u *User) Save(accountFile string) error {
	if err := os.MkdirAll(filepath.Dir(accountFile), 0700); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(u, "", "  ")
	if err := os.WriteFile(accountFile, data, 0600); err != nil {
		return err
	}
	ecKey, ok := u.key.(*ecdsa.PrivateKey)
	if !ok {
		return fmt.Errorf("不支持的密鑰類型")
	}
	keyBytes, _ := x509.MarshalECPrivateKey(ecKey)
	pemBlock := &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}
	f, err := os.OpenFile(accountFile+".key", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, pemBlock)
}
