package internal

import (
	"crypto/rand"
	"errors"
)

func GenerateSessionToken(size int) (string, error) {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLKMNOPQRSTVWXYZ0123456789"
	b := make([]byte, size)
	r, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	if r != size {
		return "", errors.New("could not generate session token")
	}

	for i := range b {
		b[i] = chars[b[i]%byte(len(chars))]
	}

	return string(b), nil
}
