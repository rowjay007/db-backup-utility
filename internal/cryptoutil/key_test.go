package cryptoutil

import (
	"encoding/base64"
	"testing"
)

func TestParseKeyBase64(t *testing.T) {
	key := make([]byte, 32)
	encoded := base64.StdEncoding.EncodeToString(key)
	parsed, err := ParseKey(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parsed) != 32 {
		t.Fatalf("unexpected key length: %d", len(parsed))
	}
}
