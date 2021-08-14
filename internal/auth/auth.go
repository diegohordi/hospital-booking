// Package auth contains handlers, services and models used to manage authentication
// and authorization.
package auth

import "golang.org/x/crypto/bcrypt"

// EncryptPassword encrypts a given string.
func EncryptPassword(pass string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// ComparePasswords compares a given encrypted password and a string, in order to check
// their equivalences.
func ComparePasswords(hashedPass, plainPass string) bool {
	byteHash := []byte(hashedPass)
	err := bcrypt.CompareHashAndPassword(byteHash, []byte(plainPass))
	return err == nil
}
