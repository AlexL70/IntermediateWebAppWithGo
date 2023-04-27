package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

type Encryption struct {
	Key []byte
}

func (e *Encryption) Encrypt(text string) (string, error) {
	const errFmtStr = "error encrypting text %w"
	binaryData := []byte(text)
	block, err := aes.NewCipher(e.Key)
	if err != nil {
		return "", fmt.Errorf(errFmtStr, err)
	}
	cipherText := make([]byte, aes.BlockSize+len(binaryData))
	iv := cipherText[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf(errFmtStr, err)
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], binaryData)
	return base64.URLEncoding.EncodeToString(cipherText), nil
}

func (e *Encryption) Decrypt(encrypted string) (string, error) {
	const errFmtStr = "error decrypting text %w"
	cipherText, err := base64.URLEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf(errFmtStr, err)
	}
	block, err := aes.NewCipher(e.Key)
	if err != nil {
		return "", fmt.Errorf(errFmtStr, err)
	}

	if len(cipherText) <= aes.BlockSize {
		return "", fmt.Errorf(errFmtStr, errors.New("input data is too small"))
	}
	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherText, cipherText)
	return string(cipherText), nil
}
