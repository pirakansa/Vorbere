package manifest

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
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
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download failed: %s status=%d", src.URL, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
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
	if algorithm != DigestAlgorithmBLAKE3 {
		return fmt.Errorf("unsupported checksum algorithm %q", algorithm)
	}
	if shared.BLAKE3Hex(content) != digest {
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
