package proxy

import (
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
	clientPool  map[int]*http.Client
	transportPool map[int]*http.Transport
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
			proxies:       cfg.Proxy.Proxies,
			occupiedMap:   occupiedMap,
			clientPool:    make(map[int]*http.Client),
			transportPool: make(map[int]*http.Transport),
			baseTimeout:   60 * time.Second,
		}
	})
	return manager
}

func (m *Manager) GetProxiedClient() (*http.Client, int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.proxies) == 0 {
		logger.GetSugaredLogger().Warn("No proxies available, using direct connection")
		return m.getOrCreateClient(0), 0
	}

	proxy := m.getNextWorkingProxy()
	if proxy == nil {
		logger.GetSugaredLogger().Warn("No working proxies available, using direct connection")
		return m.getOrCreateClient(0), 0
	}
	m.HoldProxy(proxy.ID)

	logger.GetSugaredLogger().Infof("Using proxy: %d", proxy.ID)
	return m.getOrCreateClient(proxy.ID), proxy.ID
}

func (m *Manager) getOrCreateClient(proxyID int) *http.Client {
	if client, exists := m.clientPool[proxyID]; exists {
		return client
	}

	transport := m.getOrCreateTransport(proxyID)
	client := &http.Client{
		Timeout:   m.baseTimeout,
		Transport: transport,
	}
	m.clientPool[proxyID] = client
	return client
}

func (m *Manager) getOrCreateTransport(proxyID int) *http.Transport {
	if transport, exists := m.transportPool[proxyID]; exists {
		return transport
	}

	transport := m.getBaseTransport()

	if proxyID > 0 {
		for _, proxy := range m.proxies {
			if proxy.ID == proxyID {
				proxyURL, err := url.Parse(proxy.URL)
				if err != nil {
					logger.GetSugaredLogger().Errorf("Failed to parse proxy URL for ID %d: %v", proxyID, err)
					break
				}
				transport.Proxy = http.ProxyURL(proxyURL)
				break
			}
		}
	}

	m.transportPool[proxyID] = transport
	return transport
}

func (m *Manager) ReleaseProxy(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.occupiedMap[id] = false
}

func (m *Manager) HoldProxy(id int) {
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
	for i := range len(m.proxies) {
		if !m.occupiedMap[m.proxies[i].ID] {
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
		MaxIdleConnsPerHost:   10,
		MaxConnsPerHost:       20,
	}
}

func (m *Manager) CleanupIdleConnections() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, transport := range m.transportPool {
		transport.CloseIdleConnections()
	}
}
