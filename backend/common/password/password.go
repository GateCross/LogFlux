// Package password 提供密码哈希、校验与随机生成能力，供 HTTP 接口与 CLI 复用。
package password

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinLength 是密码最小长度。
	MinLength = 8
	// MaxLength 受 bcrypt 72 字节上限约束。
	MaxLength = 72
	// DefaultLength 是随机密码的默认长度。
	DefaultLength = 16
)

// ErrInvalidPassword 表示密码不满足复杂度要求。
var ErrInvalidPassword = fmt.Errorf("密码需为 %d-%d 位，且同时包含字母和数字", MinLength, MaxLength)

// Hash 使用 bcrypt 生成密码哈希。
func Hash(plain string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// Verify 校验明文密码与哈希是否匹配。
func Verify(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}

// Validate 校验密码复杂度：长度合规，且同时包含字母与数字。
func Validate(plain string) error {
	if len(plain) < MinLength || len(plain) > MaxLength {
		return ErrInvalidPassword
	}

	hasLetter := strings.IndexFunc(plain, func(r rune) bool {
		return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
	}) >= 0
	hasDigit := strings.IndexFunc(plain, func(r rune) bool {
		return r >= '0' && r <= '9'
	}) >= 0

	if !hasLetter || !hasDigit {
		return ErrInvalidPassword
	}
	return nil
}

// Generate 生成满足复杂度的随机密码。
func Generate(length int) (string, error) {
	if length < MinLength {
		length = MinLength
	}
	if length > MaxLength {
		length = MaxLength
	}

	const (
		letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digits  = "0123456789"
	)
	all := letters + digits

	chars := make([]byte, 0, length)
	// 先各取一位字母和数字，保证两类字符必然出现
	for _, charset := range []string{letters, digits} {
		char, err := randomChar(charset)
		if err != nil {
			return "", err
		}
		chars = append(chars, char)
	}
	for len(chars) < length {
		char, err := randomChar(all)
		if err != nil {
			return "", err
		}
		chars = append(chars, char)
	}

	if err := shuffle(chars); err != nil {
		return "", err
	}

	generated := string(chars)
	if err := Validate(generated); err != nil {
		return "", errors.New("生成的密码不满足复杂度要求")
	}
	return generated, nil
}

func randomChar(charset string) (byte, error) {
	idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
	if err != nil {
		return 0, err
	}
	return charset[idx.Int64()], nil
}

func shuffle(data []byte) error {
	for i := len(data) - 1; i > 0; i-- {
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		j := int(randomIndex.Int64())
		data[i], data[j] = data[j], data[i]
	}
	return nil
}
