package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
)

func encrypt(rsaPublicKey string, content string) (string, error) {
	if rsaPublicKey == "" {
		return content, nil
	}

	pubKey, err := base64.StdEncoding.DecodeString(rsaPublicKey)
	if err != nil {
		return "", err
	}

	block, _ := pem.Decode([]byte(pubKey))
	if block == nil {
		return "", errors.New("failed to parse PEM block containing the public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", err
	}

	encryptedBytes, err := rsa.EncryptOAEP(
		sha256.New(),
		rand.Reader,
		pub.(*rsa.PublicKey),
		[]byte(content),
		nil)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encryptedBytes), nil
}

func decrypt(rsaPrivateKey string, encryptBytes string) (string, error) {
	if rsaPrivateKey == "" {
		return encryptBytes, nil
	}

	prvKey, err := base64.StdEncoding.DecodeString(rsaPrivateKey)
	if err != nil {
		return "", err
	}
	block, _ := pem.Decode([]byte(prvKey))
	if block == nil {
		return "", errors.New("failed to parse PEM block containing the private key")
	}

	prv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encryptBytes)
	if err != nil {
		return "", err
	}

	decryptedBytes, err := rsa.DecryptOAEP(
		sha256.New(),
		rand.Reader,
		prv,
		[]byte(ciphertext),
		nil,
	)
	if err != nil {
		return "", err
	}
	return string(decryptedBytes), nil
}
