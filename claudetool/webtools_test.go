package claudetool

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"testing"

	"shelley.exe.dev/llm"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(status int, body string, headers map[string]string) *http.Response {
	h := make(http.Header)
	for k, v := range headers {
		h.Set(k, v)
	}
	return &http.Response{StatusCode: status, Header: h, Body: io.NopCloser(strings.NewReader(body)), Request: &http.Request{}}
}
func outputText(out llm.ToolOut) string {
	if out.Error != nil {
		return out.Error.Error()
	}
	if len(out.LLMContent) == 0 {
		return ""
	}
	return out.LLMContent[0].Text
}

func TestBraveSearchUsesFixedIntegrationWithoutForwardingCredentials(t *testing.T) {
	var gotQuery string
	tools := &webTools{braveURL: "https://brave.int.example.test/res/v1/web/search", client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotQuery = r.URL.Query().Get("q")
		if r.URL.Host != "brave.int.example.test" || r.URL.Path != braveSearchPath {
			t.Fatalf("request URL = %s", r.URL)
		}
		if r.Header.Get("X-Subscription-Token") != "" || r.Header.Get("Authorization") != "" {
			t.Fatal("Shelley must not send credentials; exe.dev integration proxy injects them")
		}
		return response(200, `{"web":{"results":[{"title":"Example","url":"https://example.org/","description":"Snippet"}]}}`, nil), nil
	})}}
	out := tools.search(t.Context(), webSearchInput{Query: "  example query  "})
	if out.Error != nil {
		t.Fatal(out.Error)
	}
	if gotQuery != "example query" {
		t.Fatalf("query = %q", gotQuery)
	}
	if !strings.Contains(outputText(out), `"title":"Example"`) {
		t.Fatalf("output = %q", outputText(out))
	}
}

func TestBraveSearchDoesNotFollowRedirects(t *testing.T) {
	requests := 0
	tools := &webTools{braveURL: "https://brave.int.example.test/res/v1/web/search", client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		if r.URL.Host != "brave.int.example.test" {
			t.Fatalf("request went to unexpected host %q", r.URL.Host)
		}
		return response(http.StatusFound, "", map[string]string{"Location": "https://elsewhere.example/search?token=secret"}), nil
	})}}
	out := tools.search(t.Context(), webSearchInput{Query: "query"})
	if out.Error == nil || !strings.Contains(out.Error.Error(), "redirects are not followed") || strings.Contains(out.Error.Error(), "secret") {
		t.Fatalf("redirect error = %v", out.Error)
	}
	if requests != 1 {
		t.Fatalf("request count = %d, want only fixed integration host", requests)
	}
}

func TestBraveSearchIntegrationErrorsAndCancellation(t *testing.T) {
	if got := outputText((&webTools{}).search(t.Context(), webSearchInput{Query: "test"})); !strings.Contains(got, "attach the personal Brave integration") {
		t.Fatalf("missing integration error = %q", got)
	}
	tools := &webTools{braveURL: "https://brave.int.example.test/res/v1/web/search", client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return response(401, "credential-redacted", nil), nil })}}
	out := tools.search(t.Context(), webSearchInput{Query: "test"})
	if out.Error == nil || strings.Contains(out.Error.Error(), "credential-redacted") {
		t.Fatalf("integration error leaked response: %v", out.Error)
	}
	tools.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("upstream failed for " + r.URL.String())
	})}
	out = tools.search(t.Context(), webSearchInput{Query: "token secret"})
	if out.Error == nil || !strings.Contains(out.Error.Error(), "upstream failed") || strings.Contains(out.Error.Error(), "token+secret") {
		t.Fatalf("redacted request failure = %v", out.Error)
	}
	tools.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) { return nil, r.Context().Err() })}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	out = tools.search(ctx, webSearchInput{Query: "test"})
	if out.Error == nil || !strings.Contains(out.Error.Error(), "context canceled") {
		t.Fatalf("cancel error = %v", out.Error)
	}
}

func TestSafePublicURL(t *testing.T) {
	for _, raw := range []string{"file:///etc/passwd", "http://user:pass@example.org/", "http://127.0.0.1/", "http://169.254.169.254/", "http://[::1]/", "http://example.org:8080/", "https://example.org:80/", "https://brave.int.exe.xyz/", "https://notion.team.exe.cloud/"} {
		if _, err := safePublicURL(raw); err == nil {
			t.Errorf("accepted unsafe URL %q", raw)
		}
	}
	if _, err := safePublicURL("https://example.org/a?token=secret"); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		ip     string
		public bool
	}{
		{"127.0.0.1", false}, {"10.0.0.1", false}, {"169.254.169.254", false},
		{"192.168.1.1", false}, {"::1", false}, {"fc00::1", false},
		{"100.64.0.1", false}, {"192.0.0.1", false}, {"192.0.2.1", false},
		{"198.18.0.1", false}, {"198.51.100.1", false}, {"203.0.113.1", false},
		{"240.0.0.1", false}, {"::ffff:169.254.169.254", false}, {"::ffff:8.8.8.8", true},
		{"64:ff9b::a9fe:a9fe", false}, {"64:ff9b:1::a9fe:a9fe", false},
		{"2002:a9fe:a9fe::1", false}, {"2001:0000:a9fe:a9fe::1", false},
		{"::a9fe:a9fe", false}, {"2606:4700:4700:0:0:5efe:a9fe:a9fe", false},
		{"2001:db8::1", false}, {"3fff::1", false},
		{"2606:4700:4700::1111", true}, {"8.8.8.8", true},
	} {
		ip := netip.MustParseAddr(tt.ip)
		if got := publicIP(ip); got != tt.public {
			t.Errorf("publicIP(%s) = %v, want %v", tt.ip, got, tt.public)
		}
	}
}

func TestWebFetchExtractionBoundsAndRedirects(t *testing.T) {
	tools := &webTools{client: &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/redirect" {
			return response(302, "", map[string]string{"Location": "http://127.0.0.1/admin"}), nil
		}
		return response(200, `<html><head><title>Example</title><script>secret()</script></head><body><h1>Hello</h1><p>world</p></body></html>`, map[string]string{"Content-Type": "text/html"}), nil
	})}}
	out := tools.fetch(t.Context(), webFetchInput{URL: "https://example.org/page?private=secret"})
	text := outputText(out)
	if out.Error != nil || !strings.Contains(text, "Hello world") || strings.Contains(text, "secret()") || strings.Contains(text, "private=secret") {
		t.Fatalf("fetch output = %q, error=%v", text, out.Error)
	}
	out = tools.fetch(t.Context(), webFetchInput{URL: "https://example.org/redirect"})
	if out.Error == nil || !strings.Contains(out.Error.Error(), "redirect rejected") {
		t.Fatalf("unsafe redirect error = %v", out.Error)
	}
	tools.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(200, strings.Repeat("x", maxFetchBytes+1), map[string]string{"Content-Type": "text/plain"}), nil
	})}
	out = tools.fetch(t.Context(), webFetchInput{URL: "https://example.org/large"})
	if out.Error == nil || !strings.Contains(out.Error.Error(), "exceeded 1 MiB") {
		t.Fatalf("oversize error = %v", out.Error)
	}
	tools.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(200, strings.Repeat("x", maxFetchText+100), map[string]string{"Content-Type": "text/plain"}), nil
	})}
	out = tools.fetch(t.Context(), webFetchInput{URL: "https://example.org/long"})
	var page fetchedPage
	if err := json.Unmarshal([]byte(outputText(out)), &page); err != nil || len([]rune(page.Text)) != maxFetchText || !page.Truncated || !strings.HasSuffix(page.Text, "x") {
		t.Fatalf("bounded output length = %d, truncated = %v, error = %v", len([]rune(page.Text)), page.Truncated, err)
	}
	tools.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(200, strings.Repeat("é", maxFetchText-1), map[string]string{"Content-Type": "text/plain"}), nil
	})}
	out = tools.fetch(t.Context(), webFetchInput{URL: "https://example.org/multibyte"})
	if err := json.Unmarshal([]byte(outputText(out)), &page); err != nil || len([]rune(page.Text)) != maxFetchText-1 || page.Truncated || !strings.HasSuffix(page.Text, "é") {
		t.Fatalf("short multibyte output length = %d, truncated = %v, error = %v", len([]rune(page.Text)), page.Truncated, err)
	}
	tools.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(200, strings.Repeat("é", maxFetchText+1), map[string]string{"Content-Type": "text/plain"}), nil
	})}
	out = tools.fetch(t.Context(), webFetchInput{URL: "https://example.org/multibyte-long"})
	if err := json.Unmarshal([]byte(outputText(out)), &page); err != nil || len([]rune(page.Text)) != maxFetchText || !page.Truncated || !strings.HasSuffix(page.Text, "é") {
		t.Fatalf("multibyte capped output length = %d, truncated = %v, error = %v", len([]rune(page.Text)), page.Truncated, err)
	}
}

func TestWebFetchRejectsPrivateDNSResults(t *testing.T) {
	for _, resolved := range [][]netip.Addr{{netip.MustParseAddr("127.0.0.1")}, {netip.MustParseAddr("93.184.216.34"), netip.MustParseAddr("10.0.0.1")}} {
		tools := &webTools{lookup: func(context.Context, string) ([]netip.Addr, error) { return resolved, nil }}
		out := tools.fetch(t.Context(), webFetchInput{URL: "https://example.org/"})
		if out.Error == nil || !strings.Contains(out.Error.Error(), "non-public") {
			t.Fatalf("private DNS result error = %v", out.Error)
		}
	}
}

func TestWebFetchCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	tools := &webTools{client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) { return nil, r.Context().Err() })}}
	out := tools.fetch(ctx, webFetchInput{URL: "https://example.org/"})
	if out.Error == nil || !strings.Contains(out.Error.Error(), "context canceled") {
		t.Fatalf("cancel error = %v", out.Error)
	}
}
