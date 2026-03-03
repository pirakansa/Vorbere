package manifest

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pirakansa/vorbere/internal/cli/shared"
)

func download(src Source) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, src.URL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range src.Headers {
		req.Header.Set(k, v)
	}
	resp, err := newDownloadClient(src.Headers).Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"download failed: %s",
			maskHeaderValues(err.Error(), src.Headers),
		)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download failed: %s status=%d", src.URL, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func newDownloadClient(headers map[string]string) *http.Client {
	base := http.DefaultClient
	if base == nil {
		base = &http.Client{}
	}
	client := *base
	prevCheckRedirect := base.CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 0 && !sameHost(req.URL, via[len(via)-1].URL) {
			for key := range headers {
				req.Header.Del(key)
			}
		}
		if prevCheckRedirect != nil {
			return prevCheckRedirect(req, via)
		}
		return nil
	}
	return &client
}

func sameHost(left, right *url.URL) bool {
	if left == nil || right == nil {
		return false
	}
	return strings.EqualFold(left.Host, right.Host)
}

func maskHeaderValues(message string, headers map[string]string) string {
	masked := message
	for _, value := range headers {
		if value == "" {
			continue
		}
		masked = strings.ReplaceAll(masked, value, "***")
	}
	return masked
}

func verifyChecksum(content []byte, checksum string) error {
	if checksum == "" {
		return nil
	}
	algorithm, digest, err := parseChecksumSpec(checksum)
	if err != nil {
		return err
	}
	if algorithm == "" {
		return nil
	}
	computed, err := computeDigest(content, algorithm)
	if err != nil {
		return err
	}
	if computed != digest {
		return errors.New("checksum mismatch")
	}
	return nil
}

func parseChecksumSpec(value string) (string, string, error) {
	raw := strings.TrimSpace(strings.ToLower(value))
	if raw == "" {
		return "", "", nil
	}
	algorithm, digest, ok := strings.Cut(raw, ":")
	if !ok || strings.TrimSpace(algorithm) == "" || strings.TrimSpace(digest) == "" {
		return "", "", fmt.Errorf("invalid checksum format %q", value)
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return "", "", fmt.Errorf("invalid checksum hex %q", value)
	}
	return algorithm, digest, nil
}

func computeDigest(content []byte, algorithm string) (string, error) {
	switch algorithm {
	case DigestAlgorithmBLAKE3:
		return shared.BLAKE3Hex(content), nil
	case DigestAlgorithmSHA256:
		return shared.SHA256Hex(content), nil
	case DigestAlgorithmMD5:
		return shared.MD5Hex(content), nil
	default:
		return "", fmt.Errorf("unsupported checksum algorithm %q", algorithm)
	}
}
