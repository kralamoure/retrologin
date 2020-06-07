package d1login

import (
	crand "crypto/rand"
	"errors"
	"regexp"
	"strings"
)

func DecryptedPassword(encryptedPassword, key string) (string, error) {
	if key == "" {
		return "", errors.New("key is empty")
	}
	if len(encryptedPassword)%2 != 0 ||
		!regexp.MustCompile(`^[a-zA-Z\d\-_]{2,64}$`).MatchString(encryptedPassword) {
		return "", errors.New("the encrypted password is malformed")
	}

	hash := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"

	hashMap := make(map[rune]rune)
	for i, v := range hash {
		hashMap[v] = rune(i)
	}

	var pPass, pKey rune
	var aPass, aKey, anb, anb2, sum1, sum2 int

	sb := &strings.Builder{}

	for i := 0; i < len(encryptedPassword); i += 2 {
		pKey = rune(key[i/2])
		anb = int(hashMap[rune(encryptedPassword[i])])
		anb2 = int(hashMap[rune(encryptedPassword[i+1])])
		sum1 = anb + len(hash)
		sum2 = anb2 + len(hash)

		aPass = sum1 - int(pKey)
		if aPass < 0 {
			aPass += len(hash)
		}
		aPass *= 16

		aKey = sum2 - int(pKey)
		if aKey < 0 {
			aKey += len(hash)
		}

		pPass = rune(aPass + aKey)

		sb.WriteRune(pPass)
	}

	return sb.String(), nil
}

func RandomSalt(n int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyz"

	p, err := randomBytes(n)
	if err != nil {
		return "", err
	}

	for i, v := range p {
		p[i] = charset[v%byte(len(charset))]
	}

	return string(p), nil
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)

	_, err := crand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}
