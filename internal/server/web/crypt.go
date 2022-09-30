package web

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
)

var emulatedError string

// emulateError используется для Эмуляции ошибок в тесте, для тех функций, в которых невозможно замокать интерфейс
func emulateError(err error, pos int) error {
	if err != nil {
		return err
	}
	if len(emulatedError) != 0 && strings.Contains(emulatedError, fmt.Sprint(pos)) {
		return errors.New(emulatedError)
	}
	return nil
}

func GenerateCryptoKeys(keyPath string) (err error) {
	// создаём новый приватный RSA-ключ длиной 4096 бит
	// обратите внимание, что для генерации ключа и сертификата
	// используется rand.Reader в качестве источника случайных данных
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err = emulateError(err, 1); err != nil {
		return err
	}

	// кодируем сертификат и ключ в формате PEM, который
	// используется для хранения и обмена криптографическими ключами
	var publicKeyPEM bytes.Buffer
	err = pem.Encode(&publicKeyPEM, &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey),
	})
	if err = emulateError(err, 2); err != nil {
		return err
	}
	var privateKeyPEM bytes.Buffer
	err = pem.Encode(&privateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err = emulateError(err, 3); err != nil {
		return err
	}
	err = os.WriteFile(keyPath, privateKeyPEM.Bytes(), 0700)
	if err != nil {
		return err
	}
	err = os.WriteFile(strings.ReplaceAll(keyPath, ".pem", "")+".pub", publicKeyPEM.Bytes(), 0700)
	if err = emulateError(err, 4); err != nil {
		return err
	}
	return nil
}
