package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

func ParsePrivateKey(raw string) (*rsa.PrivateKey, error) {
	raw = strings.TrimSpace(raw)

	der, err := tryDecodeDER(raw)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	// PKCS#8 (openssl genpkey -outform DER)
	if k, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		if rk, ok := k.(*rsa.PrivateKey); ok {
			return rk, nil
		}
		return nil, errors.New("private key is not RSA")
	}

	// PKCS#1 (openssl genrsa)
	if rk, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return rk, nil
	}

	return nil, errors.New("failed to parse private key: unsupported format (expected PKCS#8 or PKCS#1 DER, base64-encoded)")
}

func tryDecodeDER(raw string) ([]byte, error) {
	// PEM-encoded key
	if strings.Contains(raw, "-----BEGIN") {
		block, _ := pem.Decode([]byte(raw))
		if block == nil {
			return nil, errors.New("failed to decode PEM block")
		}
		return block.Bytes, nil
	}

	// Base64 with possible line breaks
	cleaned := strings.NewReplacer("\n", "", "\r", "", " ", "").Replace(raw)

	if der, err := base64.StdEncoding.DecodeString(cleaned); err == nil {
		return der, nil
	}
	if der, err := base64.RawStdEncoding.DecodeString(cleaned); err == nil {
		return der, nil
	}
	return nil, fmt.Errorf("not valid base64")
}

func decryptRsaOaep(priv *rsa.PrivateKey, encAesKey []byte) ([]byte, error) {
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, encAesKey, nil)
}
