package proxy

import (
	"testing"
)

func TestProxyPool(t *testing.T) {
	proxies := []string{"http://proxy1:8080", "socks5://proxy2:1080"}
	pool := NewPool(proxies)

	if pool.Size() != 2 {
		t.Fatalf("expected 2 proxies, got %d", pool.Size())
	}

	if pool.Current() != "http://proxy1:8080" {
		t.Errorf("expected proxy1, got %s", pool.Current())
	}

	next := pool.Next()
	if next != "socks5://proxy2:1080" {
		t.Errorf("expected proxy2, got %s", next)
	}

	wrap := pool.Next()
	if wrap != "http://proxy1:8080" {
		t.Errorf("expected proxy1 after wrap, got %s", wrap)
	}
}
