package proxy

import (
	"strings"
	"sync"
)

// Pool manages rotating HTTP/SOCKS5 proxies
type Pool struct {
	mu      sync.Mutex
	proxies []string
	index   int
}

// NewPool creates a proxy pool filtering empty entries
func NewPool(proxies []string) *Pool {
	var cleaned []string
	for _, p := range proxies {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return &Pool{
		proxies: cleaned,
		index:   0,
	}
}

// Current returns current proxy or empty string if pool is empty
func (p *Pool) Current() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.proxies) == 0 {
		return ""
	}
	return p.proxies[p.index]
}

// Next advances to the next proxy in the list
func (p *Pool) Next() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.proxies) == 0 {
		return ""
	}

	p.index = (p.index + 1) % len(p.proxies)
	return p.proxies[p.index]
}

// Size returns the count of proxies
func (p *Pool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.proxies)
}
