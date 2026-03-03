package manifest

import (
	"fmt"
	"net/http"
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
