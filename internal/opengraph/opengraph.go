package opengraph

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lufeed/feed-parser-api/internal/logger"
	"go.uber.org/zap"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

type WebsiteInformation struct {
	Image       string
	Description string
	Icon        string
	Title       string
	HTML        string
}

type Extractor struct {
	cl      *http.Client
	baseUrl string
	host    string
	icon    bool
}

func NewExtractor(cl *http.Client, baseUrl string, host string, icon bool) *Extractor {
	return &Extractor{cl: cl, baseUrl: baseUrl, host: host, icon: icon}
}

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

func (e *Extractor) applyBrowserHeaders(req *http.Request, attempt int) {
	if len(browserHeaderProfiles) == 0 {
		return
	}
	idx := attempt % len(browserHeaderProfiles)
	for k, v := range browserHeaderProfiles[idx] {
		req.Header.Set(k, v)
	}
}

func (e *Extractor) Exec() (WebsiteInformation, error) {
	doc, err := e.getDoc()
	if err != nil {
		return WebsiteInformation{}, err
	}

	wsi := WebsiteInformation{
		Image:       e.getImage(doc),
		Description: e.getDescription(doc),
		Title:       e.getTitle(doc),
		HTML:        e.getHTML(doc),
	}

	if e.icon {
		wsi.Icon = e.getIcon(doc)
	}

	parsedHomeURL, err := url.Parse(e.baseUrl)
	if err == nil {
		newUrl := fmt.Sprintf("%s://%s", parsedHomeURL.Scheme, parsedHomeURL.Host)
		if newUrl != e.baseUrl {
			e.baseUrl = newUrl
			newDoc, err := e.getDoc()
			if err != nil {
				return wsi, nil
			}
			if wsi.Image == "" {
				wsi.Image = e.getImage(newDoc)
			}
			if e.icon && wsi.Icon == "" {
				wsi.Icon = e.getIcon(newDoc)
			}
			if wsi.Description == "" {
				wsi.Description = e.getDescription(newDoc)
			}
			if wsi.Title == "" {
				wsi.Title = e.getTitle(newDoc)
			}
		}
	}

	return wsi, err
}

func (e *Extractor) getDoc() (*html.Node, error) {
	// Clean and validate URL format
	baseUrl := strings.TrimSpace(e.baseUrl)

	// Skip empty URLs
	if baseUrl == "" {
		return nil, fmt.Errorf("empty URL provided")
	}

	// Fix protocol if needed
	if !strings.Contains(baseUrl, "://") {
		baseUrl = strings.Replace(baseUrl, ":/", "://", 1)
	}

	if strings.HasPrefix(baseUrl, "/") || strings.HasPrefix(baseUrl, "//") {
		parsedHost, err := url.Parse(e.host)
		if err == nil && parsedHost.Scheme != "" && parsedHost.Host != "" {
			if strings.HasPrefix(baseUrl, "//") {
				baseUrl = parsedHost.Scheme + ":" + baseUrl
			} else {
				baseUrl = parsedHost.Scheme + "://" + parsedHost.Host + baseUrl
			}
		} else {
			return nil, fmt.Errorf("cannot resolve relative URL with invalid host: %s", e.host)
		}
	}

	// Ensure URL has a valid scheme
	if !strings.HasPrefix(baseUrl, "http://") && !strings.HasPrefix(baseUrl, "https://") {
		// Add https:// as default scheme
		baseUrl = "https://" + baseUrl
	}

	// Validate final URL format
	parsedURL, err := url.Parse(baseUrl)
	if err != nil {
		logger.GetSugaredLogger().Warnf("Invalid URL format: %s, error: %s", baseUrl, err.Error())
		return nil, err
	}

	// Ensure URL has a host
	if parsedURL.Host == "" {
		return nil, fmt.Errorf("URL missing host: %s", baseUrl)
	}

	// Retry mechanism for fetching the URL (handles transient errors and 429)
	const maxRetries = 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Create a new request with headers per attempt
		req, err := http.NewRequest("GET", baseUrl, nil)
		if err != nil {
			logger.GetSugaredLogger().Warnf("Error creating request for %s: %s", baseUrl, err.Error())
			return nil, err
		}

		e.applyBrowserHeaders(req, rand.Intn(len(browserHeaderProfiles)))

		resp, err := e.cl.Do(req)
		if err != nil {
			// Network/transport error: retry with backoff if attempts remain
			if attempt < maxRetries-1 {
				backoff := time.Duration(math.Pow(2, float64(attempt+1))) * time.Second
				jitter := time.Duration(rand.Int63n(int64(backoff) / 2))
				retryAfter := backoff + jitter
				logger.GetSugaredLogger().Warnf("Fetch error for host:%s url:%s (attempt %d/%d), retrying after %v: %s", e.host, baseUrl, attempt+1, maxRetries, retryAfter, err.Error())
				time.Sleep(retryAfter)
				continue
			}
			logger.GetSugaredLogger().Warnf("Error fetching url from host:%s - url: %s - %s", e.host, baseUrl, err.Error())
			return nil, err
		}

		if resp.StatusCode == http.StatusOK {
			reader, err := charset.NewReader(resp.Body, resp.Header.Get("Content-Type"))
			resp.Body.Close()
			if err != nil {
				logger.GetSugaredLogger().Warnf("Error creating charset reader: host:%s url: %s err: %s", e.host, baseUrl, err.Error())
				return nil, err
			}
			doc, err := html.Parse(reader)
			if err != nil {
				logger.GetSugaredLogger().Warnf("Error parsing HTML: %s", err.Error())
				return nil, err
			}
			return doc, nil
		}

		// Not OK status
		if (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable) && attempt < maxRetries-1 {
			resp.Body.Close()
			backoff := time.Duration(math.Pow(2, float64(attempt+1))) * time.Second
			jitter := time.Duration(rand.Int63n(int64(backoff) / 2))
			retryAfter := backoff + jitter
			logger.GetSugaredLogger().Warnf("Got %d for host:%s url:%s (attempt %d/%d), retrying after %v", resp.StatusCode, e.host, baseUrl, attempt+1, maxRetries, retryAfter)
			time.Sleep(retryAfter)
			continue
		}

		// Close body and return error for non-OK
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			resp.Body.Close()
			return nil, fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
		}
		logger.GetSugaredLogger().Debugf("Received non-200 status code (%d) for %s", resp.StatusCode, baseUrl)
		resp.Body.Close()
		return nil, fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
	}

	// Exhausted retries
	return nil, fmt.Errorf("failed to fetch URL after retries: %s", baseUrl)
}

func (e *Extractor) getDescription(doc *html.Node) string {
	var ogDesc string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var property, content string
			for _, a := range n.Attr {
				if a.Key == "property" && a.Val == "og:description" {
					property = a.Val
				}
				if a.Key == "content" {
					content = a.Val
				}
			}
			if property == "og:description" && content != "" {
				ogDesc = content
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return ogDesc
}

func (e *Extractor) getImage(doc *html.Node) string {
	var ogImage string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var property, content string
			for _, a := range n.Attr {
				if (a.Key == "property" || a.Key == "name") && a.Val == "og:image" {
					property = a.Val
				}
				if a.Key == "content" {
					content = a.Val
				}
			}
			if property == "og:image" && content != "" {
				ogImage = content
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return ogImage
}

type iconInfo struct {
	href  string
	size  int // stores the largest dimension (width or height)
	score int
}

func (e *Extractor) getIcon(doc *html.Node) string {
	var icons []iconInfo
	// Regular expression to extract size from sizes attribute (e.g., "32x32", "48x48")
	sizeRegex := regexp.MustCompile(`(\d+)x(\d+)`)

	// walk DOM and collect icon link candidates with scoring
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			var relRaw, href, sizes, typ string
			for _, a := range n.Attr {
				switch strings.ToLower(a.Key) {
				case "rel":
					relRaw = strings.ToLower(a.Val)
				case "href":
					href = a.Val
				case "sizes":
					sizes = strings.ToLower(a.Val)
				case "type":
					typ = strings.ToLower(a.Val)
				}
			}

			if href == "" {
				goto next
			}

			// Recognize multiple rel tokens (e.g., "shortcut icon", "icon apple-touch-icon")
			relTokens := strings.Fields(relRaw)
			hasIconRel := false
			for _, t := range relTokens {
				if t == "icon" || t == "shortcut" || t == "shortcut-icon" || t == "apple-touch-icon" || t == "apple-touch-icon-precomposed" || t == "mask-icon" || t == "fluid-icon" { // common variants
					hasIconRel = true
					break
				}
			}
			if !hasIconRel && strings.Contains(relRaw, "icon") {
				// Fallback: any rel containing "icon"
				hasIconRel = true
			}
			if !hasIconRel {
				goto next
			}

			// determine base score
			score := 0
			size := 0

			// Prefer SVG/vector
			if strings.Contains(typ, "svg") || strings.HasSuffix(strings.ToLower(href), ".svg") {
				score += 2000
				size = 1000 // vectors scale well
			}

			// Parse sizes attr
			if sizes != "" {
				if sizes == "any" {
					score += 1500
					if size < 800 {
						size = 800
					}
				} else {
					matches := sizeRegex.FindStringSubmatch(sizes)
					if len(matches) >= 3 {
						width, _ := strconv.Atoi(matches[1])
						height, _ := strconv.Atoi(matches[2])
						if width > 0 && height > 0 {
							if width > height {
								size = width
							} else {
								size = height
							}
							score += size
						}
					}
				}
			}

			// Heuristic: filenames with -NxN
			if size == 0 {
				filenameMatches := sizeRegex.FindStringSubmatch(href)
				if len(filenameMatches) >= 3 {
					width, _ := strconv.Atoi(filenameMatches[1])
					size = width
					score += size
				}
			}

			if size == 0 {
				// default small score for unknown size
				size = 16
				score += 16
			}

			// apple-touch icons are often high-res
			if strings.Contains(relRaw, "apple-touch-icon") {
				score += 200
			}

			// Resolve to absolute and validate with HEAD
			abs := e.resolveURL(href)
			if abs != "" && e.checkHead(abs) {
				icons = append(icons, iconInfo{href: abs, size: size, score: score})
			}
		}
	next:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	// pick best candidate
	var best iconInfo
	for _, icon := range icons {
		if icon.score > best.score || (icon.score == best.score && icon.size > best.size) {
			best = icon
		}
	}

	if best.href != "" {
		return best.href
	}

	// Fallbacks: try common paths and validate
	// e.g. /favicon.ico, /favicon.svg, /apple-touch-icon.png, /favicon/favicon.svg
	baseURL, err := url.Parse(e.baseUrl)
	if err == nil && baseURL.Host != "" {
		candidates := []string{
			"/favicon.ico",
			"/favicon.svg",
			"/apple-touch-icon.png",
			"/apple-touch-icon-precomposed.png",
			path.Join("/favicon", "favicon.ico"),
			path.Join("/favicon", "favicon.svg"),
		}
		scheme := baseURL.Scheme
		if scheme == "" {
			scheme = "https"
		}
		for _, pth := range candidates {
			u := scheme + "://" + baseURL.Host + pth
			if e.checkHead(u) {
				return u
			}
		}
		// last resort: default /favicon.ico without HEAD check
		return scheme + "://" + baseURL.Host + "/favicon.ico"
	}

	return ""
}

func (e *Extractor) getTitle(doc *html.Node) string {
	var ogTitle string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var property, content string
			for _, a := range n.Attr {
				if a.Key == "property" && a.Val == "og:title" {
					property = a.Val
				}
				if a.Key == "content" {
					content = a.Val
				}
			}
			if property == "og:title" && content != "" {
				ogTitle = content
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return ogTitle
}

func (e *Extractor) getHTML(doc *html.Node) string {
	var htmlContent strings.Builder

	// First try to get the main content area
	mainContent := e.findMainContent(doc)
	if mainContent != nil {
		doc = mainContent
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		switch n.Type {
		case html.ElementNode:
			switch n.Data {
			case "script", "style", "noscript", "iframe":
				// Skip these elements
				return
			default:
				// Process all other elements by traversing their children
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					f(c)
				}
			}
		case html.TextNode:
			text := strings.TrimSpace(n.Data)
			if text != "" {
				htmlContent.WriteString(text)
				htmlContent.WriteString(" ")
			}
		}
	}
	f(doc)
	return strings.TrimSpace(htmlContent.String())
}

// findMainContent attempts to find the main content area of the page
func (e *Extractor) findMainContent(doc *html.Node) *html.Node {
	var mainContent *html.Node
	var maxScore int

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			score := 0

			// Score based on element type
			switch n.Data {
			case "main", "article":
				score += 10
			case "div", "section":
				score += 5
			}

			// Score based on id/class attributes
			for _, attr := range n.Attr {
				if attr.Key == "id" || attr.Key == "class" {
					value := strings.ToLower(attr.Val)
					if strings.Contains(value, "content") ||
						strings.Contains(value, "main") ||
						strings.Contains(value, "article") ||
						strings.Contains(value, "blog") ||
						strings.Contains(value, "post") {
						score += 5
					}
				}
			}

			// Score based on content length
			var textLength int
			var countText func(*html.Node)
			countText = func(n *html.Node) {
				if n.Type == html.TextNode {
					textLength += len(strings.TrimSpace(n.Data))
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					countText(c)
				}
			}
			countText(n)
			score += textLength / 100

			if score > maxScore {
				maxScore = score
				mainContent = n
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return mainContent
}

func (e *Extractor) checkHead(checkUrl string) bool {
	resultUrl := ""
	if checkUrl != "" {
		parsedURL, err := url.Parse(checkUrl)
		if err == nil {
			if !parsedURL.IsAbs() {
				// Get the base URL from the feed item's URL
				baseURL, err := url.Parse(e.baseUrl)
				if err == nil {
					// Remove any path segments after the last slash and then resolve
					basePathParts := strings.Split(baseURL.Path, "/")
					if len(basePathParts) > 1 {
						baseURL.Path = strings.Join(basePathParts[:len(basePathParts)-1], "/")
					}
					resultUrl = baseURL.ResolveReference(parsedURL).String()
				}
			} else {
				resultUrl = checkUrl
			}
		}

		// Test image URL before downloading with browser-like headers
		headReq, err := http.NewRequest("HEAD", resultUrl, nil)
		if err != nil {
			logger.GetSugaredLogger().With(zap.String("url", resultUrl)).Debugf("Error creating HEAD request: %s", err.Error())
			return false
		}
		// use a random-ish profile for HEAD checks to reduce blocking
		e.applyBrowserHeaders(headReq, rand.Intn(len(browserHeaderProfiles)))
		headResp, err := e.cl.Do(headReq)
		if err != nil {
			logger.GetSugaredLogger().With(zap.String("url", resultUrl)).Debugf("Error checking image URL: %s", err.Error())
			resultUrl = ""
		} else {
			headResp.Body.Close()
			if headResp.StatusCode != http.StatusOK {
				logger.GetSugaredLogger().With(zap.String("url", resultUrl), zap.Int("status", headResp.StatusCode)).Debug("Image URL returned non-200 status code")
				resultUrl = ""
			}
		}
	}
	return resultUrl != ""
}

// resolveURL resolves a possibly relative URL against e.baseUrl and supports protocol-relative URLs
func (e *Extractor) resolveURL(href string) string {
	if href == "" {
		return ""
	}
	// protocol-relative
	if strings.HasPrefix(href, "//") {
		baseURL, err := url.Parse(e.baseUrl)
		if err != nil {
			return ""
		}
		scheme := baseURL.Scheme
		if scheme == "" {
			scheme = "https"
		}
		return scheme + ":" + href
	}
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	if u.IsAbs() {
		return u.String()
	}
	baseURL, err := url.Parse(e.baseUrl)
	if err != nil {
		return ""
	}
	return baseURL.ResolveReference(u).String()
}
