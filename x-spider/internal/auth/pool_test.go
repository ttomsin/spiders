package auth

import (
	"testing"
)

func TestTokenPool(t *testing.T) {
	tokens := []string{"token1", "token2", "token3"}
	pool := NewTokenPool(tokens)

	if pool.Size() != 3 {
		t.Fatalf("expected pool size 3, got %d", pool.Size())
	}

	curr, err := pool.Current()
	if err != nil || curr != "token1" {
		t.Fatalf("expected token1, got %s", curr)
	}

	next, err := pool.Rotate()
	if err != nil || next != "token2" {
		t.Fatalf("expected token2, got %s", next)
	}

	next, _ = pool.Rotate()
	if next != "token3" {
		t.Fatalf("expected token3, got %s", next)
	}

	// Wraps around to token1
	next, _ = pool.Rotate()
	if next != "token1" {
		t.Fatalf("expected token1 on wrap around, got %s", next)
	}
}
