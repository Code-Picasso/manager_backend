package auth

import (
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

var (
	uppercase = regexp.MustCompile(`[A-Z]`)
	digit     = regexp.MustCompile(`[0-9]`)
	special   = regexp.MustCompile(`[~!@#$%^&*()_+\-=\[\]{}|;:,.<>?]`)
)

// HashPassword returns the bcrypt hash of a plaintext password.
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword reports whether plain matches a stored bcrypt hash.
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// StrongPasswordError returns the first password-policy violation, if any.
func StrongPasswordError(password string) string {
	if len(password) < 6 {
		return "The password must be at least 6 characters."
	}
	if !uppercase.MatchString(password) {
		return "The password must contain at least one uppercase letter."
	}
	if !digit.MatchString(password) {
		return "The password must contain at least one number."
	}
	if !special.MatchString(password) {
		return "The password must contain at least one special character."
	}
	return ""
}
