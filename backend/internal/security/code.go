package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

func NewNumericCode(length int) (string, error) {
	if length <= 0 {
		length = 6
	}
	out := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		out[i] = byte('0' + n.Int64())
	}
	return string(out), nil
}

func HashCode(email, purpose, code string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s", email, purpose, code)))
	return hex.EncodeToString(sum[:])
}
