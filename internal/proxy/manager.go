package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/lufeed/feed-parser-api/internal/config"

	"github.com/lufeed/feed-parser-api/internal/logger"
)

type Manager struct {
	proxies     []config.Proxy
	occupiedMap map[int]bool
	mu          sync.RWMutex
	baseTimeout time.Duration
}

func NewManager(cfg *config.AppConfig) *Manager {
	occupiedMap := make(map[int]bool)
	for i := range cfg.Proxy.Proxies {
		occupiedMap[i] = false
	}
	return &Manager{
		proxies:     cfg.Proxy.Proxies,
		occupiedMap: occupiedMap,
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
	m.occupiedMap[proxy.ID] = true

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

func (m *Manager) ReleaseProxy(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.occupiedMap[id] = false
}

func (m *Manager) ProxyCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.proxies)
}

func (m *Manager) getNextWorkingProxy() *config.Proxy {
	for i := 0; i < len(m.proxies); i++ {
		if !m.occupiedMap[i] {
			return &m.proxies[i]
		}
	}
	return nil
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
