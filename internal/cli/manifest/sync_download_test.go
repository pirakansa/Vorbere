package manifest

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDownloadMasksHeaderValuesInErrors(t *testing.T) {
	const secret = "super-secret-token"
	oldClient := http.DefaultClient
	t.Cleanup(func() {
		http.DefaultClient = oldClient
	})
	http.DefaultClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("request blocked: %s", req.Header.Get("Authorization"))
		}),
	}

	_, err := download(Source{
		URL: "https://example.com/file.txt",
		Headers: map[string]string{
			"Authorization": secret,
		},
	})
	if err == nil {
		t.Fatalf("expected download error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("expected secret header value to be masked, got: %v", err)
	}
	if !strings.Contains(err.Error(), "***") {
		t.Fatalf("expected masked marker in error, got: %v", err)
	}
}

func TestDownloadKeepsHeadersOnSameHostRedirect(t *testing.T) {
	const headerKey = "X-Api-Key"
	const headerValue = "local-secret"
	var observedHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start":
			http.Redirect(w, r, "/final", http.StatusFound)
		case "/final":
			observedHeader = r.Header.Get(headerKey)
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "ok")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	_, err := download(Source{
		URL: server.URL + "/start",
		Headers: map[string]string{
			headerKey: headerValue,
		},
	})
	if err != nil {
		t.Fatalf("download returned error: %v", err)
	}
	if observedHeader != headerValue {
		t.Fatalf("expected header to be preserved, got %q", observedHeader)
	}
}

func TestDownloadDropsHeadersOnCrossHostRedirect(t *testing.T) {
	const headerKey = "X-Api-Key"
	const headerValue = "local-secret"
	var observedHeader string
	redirectTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observedHeader = r.Header.Get(headerKey)
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	}))
	defer redirectTarget.Close()

	redirectSource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectTarget.URL+"/final", http.StatusFound)
	}))
	defer redirectSource.Close()

	_, err := download(Source{
		URL: redirectSource.URL + "/start",
		Headers: map[string]string{
			headerKey: headerValue,
		},
	})
	if err != nil {
		t.Fatalf("download returned error: %v", err)
	}
	if observedHeader != "" {
		t.Fatalf("expected header to be dropped on cross-host redirect, got %q", observedHeader)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
