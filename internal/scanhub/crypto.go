package scanhub

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

const encVersion = 1

type envelope struct {
	V          int    `json:"v"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

// DeriveKey derives a 32-byte AES key from the hub token.
func DeriveKey(token string) []byte {
	r := hkdf.New(sha256.New, []byte(token), []byte("goscan-scanhub-v1"), []byte("aes-gcm"))
	key := make([]byte, 32)
	_, _ = io.ReadFull(r, key)
	return key
}

func Encrypt(token string, plaintext []byte) ([]byte, error) {
	key := DeriveKey(token)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ct := gcm.Seal(nil, nonce, plaintext, nil)
	env := envelope{
		V:          encVersion,
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ct),
	}
	return json.Marshal(env)
}

func Decrypt(token string, wire []byte) ([]byte, error) {
	var env envelope
	if err := json.Unmarshal(wire, &env); err != nil {
		return nil, err
	}
	if env.V != encVersion {
		return nil, fmt.Errorf("versão encrypt inválida")
	}
	nonce, err := base64.StdEncoding.DecodeString(env.Nonce)
	if err != nil {
		return nil, err
	}
	ct, err := base64.StdEncoding.DecodeString(env.Ciphertext)
	if err != nil {
		return nil, err
	}
	key := DeriveKey(token)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ct, nil)
}
