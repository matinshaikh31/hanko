package aes_gcm

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// AESGCM is used to en-/decrypt the generated jwks with AES-GCM
type AESGCM struct {
	keys [][32]byte
}

// Construct a AES GCM encrypter/decrypter and check the keys as a prerequisite
func NewAESGCM(keys []string) (*AESGCM, error) {
	if len(keys) < 1 {
		return nil, errors.New("At least one encryption key must be provided.")
	}
	fixedKeys := [][32]byte{}
	for i, v := range keys {
		if len(v) != 32 {
			return nil, errors.New(fmt.Sprintf("Key Nr. %v has the wrong length. Is %v but needs to be 32.", i, len(v)))
		} else {
			fixedKeys = append(fixedKeys, convertKey(v))
		}
	}

	return &AESGCM{keys: fixedKeys}, nil
}

// convertKey converts strings to fixed 32byte long AES keys
func convertKey(key string) (res [32]byte) {
	copy(res[:], key[:32])
	return res
}

// NewEncryptionKey generates a random 256-bit key. It panics if the source of randomness fails.
func NewEncryptionKey() *[32]byte {
	key := [32]byte{}
	_, err := io.ReadFull(rand.Reader, key[:])
	if err != nil {
		panic(err)
	}
	return &key
}

// Encrypt encrypts some data with the first key in list and base64 encodes it for storage in database. mostly copy/pasted from https://github.com/gtank/cryptopasta/blob/master/encrypt.go
func (a *AESGCM) Encrypt(plaintext []byte) (string, error) {
	block, err := aes.NewCipher(a.keys[0][:])
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt tries to decrypt with every key in the list
func (a *AESGCM) Decrypt(ciphertext string) (plaintext []byte, err error) {
	for _, key := range a.keys {
		if plaintext, err = a.decrypt(ciphertext, key); err == nil {
			return plaintext, nil
		}
	}
	return nil, err
}

func (a *AESGCM) decrypt(ciphertext string, key [32]byte) ([]byte, error) {
	raw, err := base64.URLEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(raw) < gcm.NonceSize() {
		return nil, errors.New("malformed ciphertext")
	}

	plaintext, err := gcm.Open(nil,
		raw[:gcm.NonceSize()],
		raw[gcm.NonceSize():],
		nil,
	)

	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
