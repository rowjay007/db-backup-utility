package config

import (
	"os"

	"github.com/rowjay/db-backup-utility/internal/cryptoutil"
)

// EncryptConfigFile encrypts a config file with the provided key.
func EncryptConfigFile(inputPath, outputPath, key string) error {
	plain, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}
	parsed, err := cryptoutil.ParseKey(key)
	if err != nil {
		return err
	}
	ciphertext, err := cryptoutil.EncryptConfig(plain, parsed)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, ciphertext, 0o600)
}
