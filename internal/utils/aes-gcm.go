package utils

import (
	"crypto/aes"
	"crypto/cipher"
)

func decryptAesGcm(aesKey, iv, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, iv, data, nil)
}
