package manifest

import (
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
	value := strings.TrimSpace(strings.ToLower(checksum))
	if value == "" {
		return nil
	}
	if shared.BLAKE3Hex(content) != value {
		return errors.New("checksum mismatch")
	}
	return nil
}
