package proxy

import (
	"fmt"
	"github.com/lufeed/feed-parser-api/internal/config"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/lufeed/feed-parser-api/internal/logger"
)

type Manager struct {
	proxies     []config.Proxy
	mu          sync.RWMutex
	transport   *http.Transport
	baseTimeout time.Duration
}

func NewManager(cfg *config.AppConfig) *Manager {
	return &Manager{
		proxies:     cfg.Proxy.Proxies,
		baseTimeout: 60 * time.Second,
	}
}

func (m *Manager) GetProxiedClient() *http.Client {
	return &http.Client{
		Timeout:   m.baseTimeout,
		Transport: m.GetProxiedTransport(),
	}
}

func (m *Manager) GetProxiedTransport() *http.Transport {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.proxies) == 0 {
		logger.GetSugaredLogger().Warn("No proxies available, using direct connection")
		return m.getBaseTransport()
	}

	proxy := m.getNextWorkingProxy()
	if proxy == nil {
		logger.GetSugaredLogger().Warn("No working proxies available, using direct connection")
		return m.getBaseTransport()
	}

	proxyURL, err := url.Parse(fmt.Sprintf("http://%s:%s@%s:%s",
		proxy.Username, proxy.Password, proxy.Address, proxy.Port))
	if err != nil {
		logger.GetSugaredLogger().Errorf("Failed to parse proxy URL: %v", err)
		return m.getBaseTransport()
	}

	transport := m.getBaseTransport()
	transport.Proxy = http.ProxyURL(proxyURL)

	logger.GetSugaredLogger().Infof("Using proxy: %s:%s", proxy.Address, proxy.Port)

	return transport
}

func (m *Manager) getNextWorkingProxy() *config.Proxy {
	randomIdx := rand.Intn(len(m.proxies))
	return &m.proxies[randomIdx]
}

func (m *Manager) getBaseTransport() *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 60 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   60 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
	}
}
