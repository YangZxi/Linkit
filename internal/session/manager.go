package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

type entry struct {
	userID   int64
	expireAt time.Time
}

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]entry
}

func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]entry),
	}
}

func (m *Manager) Create(userID int64, ttl time.Duration) (string, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	m.sessions[sessionID] = entry{
		userID:   userID,
		expireAt: time.Now().Add(ttl),
	}
	m.mu.Unlock()
	return sessionID, nil
}

func (m *Manager) Rotate(oldSessionID string, userID int64, ttl time.Duration) (string, error) {
	sessionID, err := m.Create(userID, ttl)
	if err != nil {
		return "", err
	}
	m.Delete(oldSessionID)
	return sessionID, nil
}

func (m *Manager) Resolve(sessionID string) (int64, bool) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return 0, false
	}
	m.mu.RLock()
	item, ok := m.sessions[sessionID]
	m.mu.RUnlock()
	if !ok {
		return 0, false
	}
	if !item.expireAt.After(time.Now()) {
		m.Delete(sessionID)
		return 0, false
	}
	return item.userID, true
}

func (m *Manager) Delete(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	m.mu.Lock()
	delete(m.sessions, sessionID)
	m.mu.Unlock()
}

func (m *Manager) CleanupExpired(now time.Time) int {
	removed := 0
	m.mu.Lock()
	for sessionID, item := range m.sessions {
		if !item.expireAt.After(now) {
			delete(m.sessions, sessionID)
			removed++
		}
	}
	m.mu.Unlock()
	return removed
}

func (m *Manager) StartCleanup(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case now := <-ticker.C:
				m.CleanupExpired(now)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func generateSessionID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
