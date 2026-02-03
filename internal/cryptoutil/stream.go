package cryptoutil

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/minio/sio"
)

const (
	configMagic = "DBU1"
	configVer   = uint16(1)
)

// EncryptWriter returns a streaming encrypting writer using DARE (sio).
func EncryptWriter(w io.Writer, key []byte) (io.WriteCloser, error) {
	return sio.EncryptWriter(w, sio.Config{Key: key})
}

// DecryptReader returns a streaming decrypting reader using DARE (sio).
func DecryptReader(r io.Reader, key []byte) (io.Reader, error) {
	return sio.DecryptReader(r, sio.Config{Key: key})
}

// EncryptConfig encrypts a config payload with a small header.
func EncryptConfig(plain []byte, key []byte) ([]byte, error) {
	buf := &bytes.Buffer{}
	if _, err := buf.WriteString(configMagic); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, configVer); err != nil {
		return nil, err
	}
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	if _, err := buf.Write(nonce); err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ciphertext := aead.Seal(nil, nonce, plain, nil)
	if _, err := buf.Write(ciphertext); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecryptConfig decrypts a config payload.
func DecryptConfig(ciphertext []byte, key []byte) ([]byte, error) {
	if len(ciphertext) < 4+2+12 {
		return nil, fmt.Errorf("config cipher too short")
	}
	if string(ciphertext[:4]) != configMagic {
		return nil, fmt.Errorf("invalid config header")
	}
	ver := binary.BigEndian.Uint16(ciphertext[4:6])
	if ver != configVer {
		return nil, fmt.Errorf("unsupported config version %d", ver)
	}
	nonce := ciphertext[6:18]
	payload := ciphertext[18:]
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := aead.Open(nil, nonce, payload, nil)
	if err != nil {
		return nil, err
	}
	return plain, nil
}
