package config

import (
	"encoding/hex"
	"errors"
	"os"
)

type Config struct {
	DBEncryptionKey string
}

func Load() (*Config, error) {
	key := os.Getenv("DB_ENCRYPTION_KEY")
	if key == "" {
		return nil, errors.New("DB_ENCRYPTION_KEY environment variable is required")
	}
	if len(key) != 64 {
		return nil, errors.New("DB_ENCRYPTION_KEY must be exactly 64 hex characters (32 bytes)")
	}
	if _, err := hex.DecodeString(key); err != nil {
		return nil, errors.New("DB_ENCRYPTION_KEY must be valid hex string")
	}
	return &Config{DBEncryptionKey: key}, nil
}
