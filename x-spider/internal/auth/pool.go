package auth

import (
	"errors"
	"strings"
	"sync"
)

// TokenPool manages a thread-safe round-robin pool of Twitter auth tokens
type TokenPool struct {
	mu     sync.Mutex
	tokens []string
	index  int
}

// NewTokenPool initializes a pool with unique, non-empty tokens
func NewTokenPool(tokens []string) *TokenPool {
	seen := make(map[string]bool)
	var cleaned []string

	for _, t := range tokens {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true
			cleaned = append(cleaned, trimmed)
		}
	}

	return &TokenPool{
		tokens: cleaned,
		index:  0,
	}
}

// Current returns the current active token in the pool
func (p *TokenPool) Current() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.tokens) == 0 {
		return "", errors.New("token pool is empty")
	}
	return p.tokens[p.index], nil
}

// Rotate advances to the next token in the pool
func (p *TokenPool) Rotate() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.tokens) == 0 {
		return "", errors.New("token pool is empty")
	}

	p.index = (p.index + 1) % len(p.tokens)
	return p.tokens[p.index], nil
}

// Size returns total count of unique tokens in pool
func (p *TokenPool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.tokens)
}
