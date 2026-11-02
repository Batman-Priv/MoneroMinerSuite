package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type PanelAuth struct {
	Username string `json:"username"`
	SaltHex  string `json:"salt"`
	HashHex  string `json:"hash"`
}

func hashPassword(salt, password string) string {
	sum := sha256.Sum256([]byte(salt + ":" + password))
	return hex.EncodeToString(sum[:])
}

func NewPanelAuth(username, password string) (*PanelAuth, error) {
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return nil, err
	}
	salt := hex.EncodeToString(saltBytes)
	return &PanelAuth{
		Username: username,
		SaltHex:  salt,
		HashHex:  hashPassword(salt, password),
	}, nil
}

func LoadPanelAuth(path string) (*PanelAuth, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var a PanelAuth
	if err := json.Unmarshal(b, &a); err != nil {
		return nil, fmt.Errorf("invalid panel-auth.json file: %w", err)
	}
	return &a, nil
}

func (a *PanelAuth) Save(path string) error {
	b, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

func (a *PanelAuth) Verify(username, password string) bool {
	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(a.Username)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(hashPassword(a.SaltHex, password)), []byte(a.HashHex)) == 1
	return userOK && passOK
}

const sessionCookieName = "miner_panel_session"
const sessionLifetime = 7 * 24 * time.Hour

type SessionStore struct {
	mu       sync.Mutex
	sessions map[string]time.Time 
}

func NewSessionStore() *SessionStore {
	s := &SessionStore{sessions: make(map[string]time.Time)}
	go s.cleanupLoop()
	return s
}

func (s *SessionStore) Create() (string, time.Time) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token := hex.EncodeToString(b)
	expiry := time.Now().Add(sessionLifetime)
	s.mu.Lock()
	s.sessions[token] = expiry
	s.mu.Unlock()
	return token, expiry
}

func (s *SessionStore) Valid(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.sessions[token]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(s.sessions, token)
		return false
	}
	return true
}

func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

func (s *SessionStore) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for token, exp := range s.sessions {
			if now.After(exp) {
				delete(s.sessions, token)
			}
		}
		s.mu.Unlock()
	}
}
