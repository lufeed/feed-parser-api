package opengraph

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

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

func (e *Extractor) Exec() (WebsiteInformation, error) {
	doc, err := e.getDoc()
	if err != nil {
		return WebsiteInformation{}, err
	}

	wsi := WebsiteInformation{
		Image:       e.getImage(doc),
		Description: e.getDescription(doc),
		Title:       e.getTitle(doc),
	}

	if e.icon {
		wsi.Icon = e.getIcon(doc)
	}

	if e.checkHead(wsi.Image) || (e.icon && e.checkHead(wsi.Icon)) {
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

	// Create a new request with headers
	req, err := http.NewRequest("GET", baseUrl, nil)
	if err != nil {
		logger.GetSugaredLogger().Warnf("Error creating request for %s: %s", baseUrl, err.Error())
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36 Edg/134.0.0.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	// Make the request
	resp, err := e.cl.Do(req)
	if err != nil {
		logger.GetSugaredLogger().Warnf("Error fetching url from host:%s - url: %s - %s", e.host, baseUrl, err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	// Check if we got a successful response
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
		}
		logger.GetSugaredLogger().Debugf("Received non-200 status code (%d) for %s", resp.StatusCode, baseUrl)
		return nil, fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
	}

	reader, err := charset.NewReader(resp.Body, resp.Header.Get("Content-Type"))
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
	href string
	size int // stores the largest dimension (width or height)
}

func (e *Extractor) getIcon(doc *html.Node) string {
	var icons []iconInfo
	// Regular expression to extract size from sizes attribute (e.g., "32x32", "48x48")
	sizeRegex := regexp.MustCompile(`(\d+)x(\d+)`)

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			var rel, href, sizes string
			for _, a := range n.Attr {
				if a.Key == "rel" {
					rel = strings.ToLower(a.Val)
				}
				if a.Key == "href" {
					href = a.Val
				}
				if a.Key == "sizes" {
					sizes = a.Val
				}
			}

			// Check for favicon related link tags
			if (rel == "icon" || rel == "shortcut icon" || rel == "apple-touch-icon") && href != "" {
				size := 0

				// Parse size from the sizes attribute
				if sizes != "" {
					// Handle "any" size which is used for SVG icons - these are typically high quality
					if sizes == "any" {
						size = 1000 // Assign a high value to prefer SVG icons
					} else {
						matches := sizeRegex.FindStringSubmatch(sizes)
						if len(matches) >= 3 {
							width, _ := strconv.Atoi(matches[1])
							height, _ := strconv.Atoi(matches[2])
							if width > 0 && height > 0 {
								// Use the larger dimension
								if width > height {
									size = width
								} else {
									size = height
								}
							}
						}
					}
				}

				// Handle icons without explicit size - check filename for size hints
				if size == 0 && href != "" {
					// Look for size patterns in filenames (e.g., favicon-32x32.png)
					filenameMatches := sizeRegex.FindStringSubmatch(href)
					if len(filenameMatches) >= 3 {
						width, _ := strconv.Atoi(filenameMatches[1])
						size = width
					} else {
						// Default size for icons without size information
						size = 16
					}
				}

				icons = append(icons, iconInfo{
					href: href,
					size: size,
				})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	// Find the largest icon
	var largestIcon string
	var largestSize int

	for _, icon := range icons {
		if icon.size > largestSize {
			largestSize = icon.size
			largestIcon = icon.href
		}
	}

	// If no icon found in HTML, try the default location
	if largestIcon == "" {
		// Try to extract domain for default favicon
		baseURL, err := url.Parse(e.baseUrl)
		if err == nil && baseURL.Host != "" {
			// Use scheme from original URL or default to https
			scheme := baseURL.Scheme
			if scheme == "" {
				scheme = "https"
			}
			largestIcon = scheme + "://" + baseURL.Host + "/favicon.ico"
		}
	} else {
		// Make relative URL absolute
		iconURL, err := url.Parse(largestIcon)
		if err == nil && !iconURL.IsAbs() {
			baseURL, err := url.Parse(e.baseUrl)
			if err == nil {
				// Use the base URL's scheme and host if available
				if baseURL.Scheme != "" && baseURL.Host != "" {
					if strings.HasPrefix(largestIcon, "/") {
						largestIcon = baseURL.Scheme + "://" + baseURL.Host + largestIcon
					} else {
						// For relative URLs not starting with /, we need to consider the current path
						basePathParts := strings.Split(baseURL.Path, "/")
						basePath := ""
						if len(basePathParts) > 1 {
							basePath = strings.Join(basePathParts[:len(basePathParts)-1], "/") + "/"
						}
						largestIcon = baseURL.Scheme + "://" + baseURL.Host + basePath + largestIcon
					}
				}
			}
		}
	}

	// Validate icon URL before returning
	if largestIcon != "" {
		iconURL, err := url.Parse(largestIcon)
		if err != nil || iconURL.Scheme == "" || iconURL.Host == "" {
			logger.GetSugaredLogger().With(zap.String("icon_url", largestIcon)).Debug("Invalid icon URL")
			return ""
		}
	}

	return largestIcon
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

		// Test image URL before downloading
		headResp, err := e.cl.Head(resultUrl)
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
