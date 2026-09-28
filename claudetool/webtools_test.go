package claudetool

import (
	"context"
	"encoding/json"
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

func TestBraveSearchIntegrationErrorsAndCancellation(t *testing.T) {
	if got := outputText((&webTools{}).search(t.Context(), webSearchInput{Query: "test"})); !strings.Contains(got, "attach the personal Brave integration") {
		t.Fatalf("missing integration error = %q", got)
	}
	tools := &webTools{braveURL: "https://brave.int.example.test/res/v1/web/search", client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return response(401, "credential-redacted", nil), nil })}}
	out := tools.search(t.Context(), webSearchInput{Query: "test"})
	if out.Error == nil || strings.Contains(out.Error.Error(), "credential-redacted") {
		t.Fatalf("integration error leaked response: %v", out.Error)
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
	for _, raw := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "192.168.1.1", "::1", "fc00::1", "100.64.0.1", "192.0.0.1", "198.18.0.1", "2001:db8::1", "2606:4700:4700::1111"} {
		ip := netip.MustParseAddr(raw)
		if publicIP(ip) != (raw == "2606:4700:4700::1111") {
			t.Errorf("publicIP(%s) = %v", raw, publicIP(ip))
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
	if err := json.Unmarshal([]byte(outputText(out)), &page); err != nil || len(page.Text) != maxFetchText {
		t.Fatalf("bounded output length = %d, error = %v", len(page.Text), err)
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
