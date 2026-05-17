package utils

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
)

const (
	rsaLen       = 256
	ivLen        = 12
	gcmTagMinLen = 16
)

func Unpack[T any](enc []byte, priv *rsa.PrivateKey) (*T, error) {
	if len(enc) < rsaLen+ivLen+gcmTagMinLen {
		return nil, fmt.Errorf("encrypted data too short: %d", len(enc))
	}

	encAesKey := enc[:rsaLen]
	iv := enc[rsaLen : rsaLen+ivLen]
	encData := enc[rsaLen+ivLen:]

	aesKey, err := decryptRsaOaep(priv, encAesKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt rsa oaep: %w", err)
	}

	data, err := decryptAesGcm(aesKey, iv, encData)
	if err != nil {
		return nil, fmt.Errorf("decrypt aes gcm: %w", err)
	}

	t := new(T)
	if err := json.Unmarshal(data, t); err != nil {
		log.Printf("failed to unmarshal data: %v", err)
		return nil, err
	}
	return t, nil
}
