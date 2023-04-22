package models

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"time"
)

const (
	ScopeAuthentication = "authentication"
)

// Token is a type for authentication tokens
type Token struct {
	PlainText string    `json:"token"`
	UserID    int64     `json:"-"`
	Hash      []byte    `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

// GenerateToken generates token for user identified by userID that lasts for ttl
// and returns it or possibly an error
func GenerateToken(userID int, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: int64(userID),
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	rndBytes := make([]byte, 16)
	_, err := rand.Read(rndBytes)
	if err != nil {
		return nil, fmt.Errorf("error generating token: %w", err)
	}
	token.PlainText = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(rndBytes)
	hash := sha256.Sum256([]byte(token.PlainText))
	token.Hash = hash[:]
	return token, nil
}
