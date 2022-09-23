package web

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"strings"
)

func GenerateCryptoKeys(keyPath string) error {
	// создаём новый приватный RSA-ключ длиной 4096 бит
	// обратите внимание, что для генерации ключа и сертификата
	// используется rand.Reader в качестве источника случайных данных
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	// кодируем сертификат и ключ в формате PEM, который
	// используется для хранения и обмена криптографическими ключами
	var publicKeyPEM bytes.Buffer
	err = pem.Encode(&publicKeyPEM, &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey),
	})
	if err != nil {
		return err
	}
	var privateKeyPEM bytes.Buffer
	err = pem.Encode(&privateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err != nil {
		return err
	}
	// sum:= append(privateKeyPEM.Bytes(), certPEM.Bytes()...)
	err = os.WriteFile(keyPath, privateKeyPEM.Bytes(), 0700)
	if err != nil {
		return err
	}
	os.WriteFile(strings.ReplaceAll(keyPath, ".pem", "")+".pub", publicKeyPEM.Bytes(), 0700)
	return nil
}
