package claudetool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
	"shelley.exe.dev/exeenv"
	"shelley.exe.dev/llm"
)

const (
	braveSearchPath = "/res/v1/web/search"
	maxFetchBytes   = 1 << 20
	maxFetchText    = 20_000
)

type webSearchInput struct {
	Query string `json:"query"`
}
type webFetchInput struct {
	URL string `json:"url"`
}
type braveResponse struct {
	Web struct {
		Results []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
		} `json:"results"`
	} `json:"web"`
}
type braveResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
}
type webResults struct {
	Query   string        `json:"query"`
	Results []braveResult `json:"results"`
}
type fetchedPage struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

type webTools struct {
	braveURL string
	initErr  error
	client   *http.Client
	lookup   func(context.Context, string) ([]netip.Addr, error)
}

func newWebTools() *webTools {
	t := &webTools{lookup: func(ctx context.Context, host string) ([]netip.Addr, error) {
		return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	}}
	env, err := exeenv.Current()
	if err != nil {
		t.initErr = err
		return t
	}
	t.braveURL = env.IntegrationURL("brave", false) + braveSearchPath
	return t
}

func (t *webTools) search(ctx context.Context, in webSearchInput) llm.ToolOut {
	query := strings.TrimSpace(in.Query)
	if query == "" {
		return llm.ErrorfToolOut("query is required")
	}
	if t.initErr != nil {
		return llm.ErrorfToolOut("resolve exe.dev integration environment: %w", t.initErr)
	}
	if t.braveURL == "" {
		return llm.ErrorfToolOut("Brave Search integration is unavailable; attach the personal Brave integration to this exe.dev VM")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.braveURL, nil)
	if err != nil {
		return llm.ErrorfToolOut("create Brave search request: %w", err)
	}
	q := req.URL.Query()
	q.Set("q", query)
	q.Set("count", "8")
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/json")
	client := t.client
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second, Transport: &http.Transport{Proxy: nil}}
	}
	resp, err := client.Do(req)
	if err != nil {
		return llm.ErrorfToolOut("Brave search request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
			return llm.ErrorfToolOut("Brave Search integration returned HTTP %d; attach the personal Brave integration to this exe.dev VM", resp.StatusCode)
		}
		return llm.ErrorfToolOut("Brave Search integration returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20+1))
	if err != nil {
		return llm.ErrorfToolOut("read Brave search response: %w", err)
	}
	if len(body) > 1<<20 {
		return llm.ErrorfToolOut("Brave search response exceeded 1 MiB")
	}
	var decoded braveResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return llm.ErrorfToolOut("decode Brave search response: %w", err)
	}
	out := webResults{Query: query}
	for _, result := range decoded.Web.Results {
		if len(out.Results) == 8 {
			break
		}
		u, err := url.Parse(result.URL)
		if err != nil || u.Scheme != "https" && u.Scheme != "http" || u.User != nil {
			continue
		}
		out.Results = append(out.Results, braveResult{Title: result.Title, URL: result.URL, Description: result.Description})
	}
	return webToolOut(out)
}

func (t *webTools) fetch(ctx context.Context, in webFetchInput) llm.ToolOut {
	u, err := safePublicURL(in.URL)
	if err != nil {
		return llm.ErrorfToolOut("web_fetch rejected URL: %w", err)
	}
	client := t.safeClient()
	for redirects := 0; redirects <= 3; redirects++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return llm.ErrorfToolOut("create web_fetch request: %w", err)
		}
		req.Header.Set("User-Agent", "Shelley-WebFetch/1.0")
		req.Header.Set("Accept", "text/html,text/plain,application/xhtml+xml")
		resp, err := client.Do(req)
		if err != nil {
			return llm.ErrorfToolOut("web_fetch request failed: %w", scrubURLFromError(err, u))
		}
		if resp.StatusCode >= 300 && resp.StatusCode < 400 {
			loc := resp.Header.Get("Location")
			resp.Body.Close()
			if loc == "" {
				return llm.ErrorfToolOut("web_fetch redirect had no Location")
			}
			if redirects == 3 {
				return llm.ErrorfToolOut("web_fetch exceeded 3 redirects")
			}
			next, err := u.Parse(loc)
			if err != nil {
				return llm.ErrorfToolOut("web_fetch received an invalid redirect")
			}
			u, err = safePublicURL(next.String())
			if err != nil {
				return llm.ErrorfToolOut("web_fetch redirect rejected: %w", err)
			}
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			return llm.ErrorfToolOut("web_fetch returned HTTP %d", resp.StatusCode)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxFetchBytes+1))
		resp.Body.Close()
		if err != nil {
			return llm.ErrorfToolOut("read web_fetch response: %w", err)
		}
		if len(body) > maxFetchBytes {
			return llm.ErrorfToolOut("web_fetch response exceeded 1 MiB")
		}
		mediaType := strings.ToLower(resp.Header.Get("Content-Type"))
		if !strings.Contains(mediaType, "text/html") && !strings.Contains(mediaType, "text/plain") && !strings.Contains(mediaType, "application/xhtml+xml") {
			return llm.ErrorfToolOut("web_fetch supports HTML and plain text responses only")
		}
		text, title := extractPage(body, strings.Contains(mediaType, "html") || strings.Contains(mediaType, "xhtml"))
		if len(text) > maxFetchText {
			text = text[:maxFetchText]
		}
		return webToolOut(fetchedPage{URL: u.Scheme + "://" + u.Host + u.Path, Title: title, Text: text})
	}
	return llm.ErrorfToolOut("web_fetch exceeded redirect limit")
}

func (t *webTools) safeClient() *http.Client {
	if t.client != nil {
		return t.client
	}
	transport := &http.Transport{Proxy: nil, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 8 * time.Second, DisableKeepAlives: true}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := t.lookup(ctx, host)
		if err != nil {
			return nil, errors.New("destination DNS lookup failed")
		}
		if len(ips) == 0 {
			return nil, errors.New("destination has no addresses")
		}
		for _, ip := range ips {
			if !publicIP(ip) {
				return nil, errors.New("destination resolves to a non-public address")
			}
		}
		return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}
	return &http.Client{Transport: transport, Timeout: 12 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func safePublicURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, errors.New("invalid URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.New("only http and https are allowed")
	}
	if u.User != nil {
		return nil, errors.New("URL credentials are not allowed")
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "" {
		return nil, errors.New("URL host is required")
	}
	for _, suffix := range []string{".int.exe.xyz", ".team.exe.xyz", ".int.exe.cloud", ".team.exe.cloud"} {
		if strings.HasSuffix(host, suffix) {
			return nil, errors.New("exe.dev integration hosts are not fetchable")
		}
	}
	if p, err := netip.ParseAddr(host); err == nil && !publicIP(p) {
		return nil, errors.New("destination IP is not public")
	}
	if u.Port() != "" && (u.Scheme == "http" && u.Port() != "80" || u.Scheme == "https" && u.Port() != "443") {
		return nil, errors.New("only the standard port for the URL scheme is allowed")
	}
	return u, nil
}

func publicIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalMulticast() {
		return false
	}
	for _, prefix := range []netip.Prefix{netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("2001:db8::/32")} {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

func extractPage(body []byte, isHTML bool) (string, string) {
	if !isHTML {
		return strings.TrimSpace(string(body)), ""
	}
	root, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return "", ""
	}
	var title strings.Builder
	var text strings.Builder
	var walk func(*html.Node, bool)
	walk = func(n *html.Node, hidden bool) {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style" || n.Data == "noscript" || n.Data == "svg") {
			hidden = true
		}
		if !hidden && n.Type == html.TextNode {
			text.WriteString(n.Data)
			text.WriteByte(' ')
		}
		if n.Type == html.ElementNode && n.Data == "title" {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.TextNode {
					title.WriteString(c.Data)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, hidden)
		}
	}
	walk(root, false)
	return strings.Join(strings.Fields(text.String()), " "), strings.TrimSpace(title.String())
}

func scrubURLFromError(err error, u *url.URL) error {
	return fmt.Errorf("%s", strings.ReplaceAll(err.Error(), u.String(), "[URL]"))
}
func webToolOut(v any) llm.ToolOut {
	b, err := json.Marshal(v)
	if err != nil {
		return llm.ErrorfToolOut("encode web tool result: %w", err)
	}
	return llm.ToolOut{LLMContent: llm.TextContent(string(b))}
}

func webSearchTool(tools *webTools) *llm.Tool {
	return &llm.Tool{Name: "web_search", Description: "Search the public web with Brave Search through the exe.dev personal Brave integration. Attach that integration to the VM; credentials stay in exe.dev's integration proxy.", InputSchema: llm.MustSchema(`{"type":"object","properties":{"query":{"type":"string","description":"Search query"}},"required":["query"],"additionalProperties":false}`), Run: llm.RunJSON(tools.search)}
}
func webFetchTool(tools *webTools) *llm.Tool {
	return &llm.Tool{Name: "web_fetch", Description: "Fetch and extract readable text from a public HTTP(S) page. Rejects private/internal addresses, credentials, nonstandard ports, and unsafe redirects. HTML/plain-text only; 1 MiB response and 20,000-character output limit.", InputSchema: llm.MustSchema(`{"type":"object","properties":{"url":{"type":"string","description":"Public HTTP or HTTPS URL"}},"required":["url"],"additionalProperties":false}`), Run: llm.RunJSON(tools.fetch)}
}
