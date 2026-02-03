package cryptoutil

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// ParseKey expects a 32-byte key in base64 or hex form.
func ParseKey(key string) ([]byte, error) {
	if key == "" {
		return nil, errors.New("encryption key is empty")
	}
	trimmed := strings.TrimSpace(key)
	var data []byte
	var err error

	switch {
	case strings.HasPrefix(trimmed, "base64:"):
		data, err = base64.StdEncoding.DecodeString(strings.TrimPrefix(trimmed, "base64:"))
	case strings.HasPrefix(trimmed, "hex:"):
		data, err = hex.DecodeString(strings.TrimPrefix(trimmed, "hex:"))
	default:
		data, err = base64.StdEncoding.DecodeString(trimmed)
		if err != nil {
			data, err = hex.DecodeString(trimmed)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("decode key: %w", err)
	}
	if len(data) != 32 {
		return nil, fmt.Errorf("invalid key length: %d (expected 32 bytes)", len(data))
	}
	return data, nil
}
