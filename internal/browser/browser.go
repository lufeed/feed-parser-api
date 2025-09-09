package browser

import (
	"math/rand"
)

// A small set of realistic browser header profiles to reduce server blocking.
// We rotate these across attempts to simulate different clients.
var browserHeaderProfiles = []map[string]string{
	// Chrome on macOS (Edge-like UA kept from previous behavior)
	{
		"User-Agent":                "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36 Edg/134.0.0.0",
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Language":           "en-US,en;q=0.9",
		"Upgrade-Insecure-Requests": "1",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-User":            "?1",
		"Sec-Fetch-Dest":            "document",
		"sec-ch-ua":                 `"Chromium";v="134", "Not_A Brand";v="24", "Microsoft Edge";v="134"`,
		"sec-ch-ua-mobile":          "?0",
		"sec-ch-ua-platform":        `"macOS"`,
	},
	// Firefox on macOS
	{
		"User-Agent":                "Mozilla/5.0 (Macintosh; Intel Mac OS X 14.5; rv:129.0) Gecko/20100101 Firefox/129.0",
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Language":           "en-US,en;q=0.9",
		"Upgrade-Insecure-Requests": "1",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-User":            "?1",
		"Sec-Fetch-Dest":            "document",
	},
	// Safari on macOS
	{
		"User-Agent":                "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15",
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Language":           "en-US,en;q=0.9",
		"Upgrade-Insecure-Requests": "1",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-User":            "?1",
		"Sec-Fetch-Dest":            "document",
	},
	// Mobile Safari on iPhone
	{
		"User-Agent":         "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1",
		"Accept":             "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Language":    "en-US,en;q=0.9",
		"Sec-Fetch-Site":     "none",
		"Sec-Fetch-Mode":     "navigate",
		"Sec-Fetch-User":     "?1",
		"Sec-Fetch-Dest":     "document",
		"sec-ch-ua-mobile":   "?1",
		"sec-ch-ua-platform": `"iOS"`,
	},
	// Chrome on Windows
	{
		"User-Agent":                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36",
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Language":           "en-US,en;q=0.9",
		"Upgrade-Insecure-Requests": "1",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-User":            "?1",
		"Sec-Fetch-Dest":            "document",
		"sec-ch-ua":                 `"Chromium";v="134", "Not_A Brand";v="24", "Google Chrome";v="134"`,
		"sec-ch-ua-mobile":          "?0",
		"sec-ch-ua-platform":        `"Windows"`,
	},
}

func GetBrowserHeaders() map[string]string {
	return browserHeaderProfiles[rand.Intn(len(browserHeaderProfiles))]
}

var userAgents = []string{
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36 Edg/134.0.0.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14.5; rv:129.0) Gecko/20100101 Firefox/129.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15",
}

func GetUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}
