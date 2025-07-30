package proxy

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/lufeed/feed-parser-api/internal/config"

	"github.com/lufeed/feed-parser-api/internal/logger"
)

var (
	once    sync.Once
	manager *Manager
)

type Manager struct {
	proxies     []config.Proxy
	occupiedMap map[int]bool
	mu          sync.RWMutex
	baseTimeout time.Duration
}

func NewManager(cfg *config.AppConfig) *Manager {
	once.Do(func() {
		occupiedMap := make(map[int]bool)
		for _, p := range cfg.Proxy.Proxies {
			occupiedMap[p.ID] = false
		}
		manager = &Manager{
			proxies:     cfg.Proxy.Proxies,
			occupiedMap: occupiedMap,
			baseTimeout: 60 * time.Second,
		}
	})
	return manager
}

func (m *Manager) GetProxiedClient() (*http.Client, int) {
	transport, id := m.GetProxiedTransport()
	return &http.Client{
		Timeout:   m.baseTimeout,
		Transport: transport,
	}, id
}

func (m *Manager) GetProxiedTransport() (*http.Transport, int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.proxies) == 0 {
		logger.GetSugaredLogger().Warn("No proxies available, using direct connection")
		return m.getBaseTransport(), 0
	}

	proxy := m.getNextWorkingProxy()
	if proxy == nil {
		logger.GetSugaredLogger().Warn("No working proxies available, using direct connection")
		return m.getBaseTransport(), 0
	}
	m.HoldProxy(proxy.ID)

	proxyURL, err := url.Parse(fmt.Sprintf("http://%s:%s@%s:%s",
		proxy.Username, proxy.Password, proxy.Address, proxy.Port))
	if err != nil {
		logger.GetSugaredLogger().Errorf("Failed to parse proxy URL: %v", err)
		return m.getBaseTransport(), 0
	}

	transport := m.getBaseTransport()
	transport.Proxy = http.ProxyURL(proxyURL)

	logger.GetSugaredLogger().Infof("Using proxy: %s:%s", proxy.Address, proxy.Port)

	return transport, proxy.ID
}

func (m *Manager) ReleaseProxy(id int) {
	//m.mu.Lock()
	//defer m.mu.Unlock()
	m.occupiedMap[id] = false
}

func (m *Manager) HoldProxy(id int) {
	//m.mu.Lock()
	//defer m.mu.Unlock()
	m.occupiedMap[id] = true
}

func (m *Manager) ProxyCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return max(len(m.proxies), 1)
}

func (m *Manager) getNextWorkingProxy() *config.Proxy {
	pList := m.proxies
	rand.Shuffle(len(pList), func(i, j int) { pList[i], pList[j] = pList[j], pList[i] })
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
