package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/99designs/keyring"

	"github.com/dedene/realtime-register-cli/internal/config"
)

// ErrNoAPIKey is returned when no API key is configured.
var ErrNoAPIKey = errors.New("no API key configured")

const (
	envAPIKey          = "RR_API_KEY"
	envKeyringPassword = "RR_KEYRING_PASSWORD"
	envKeyringBackend  = "RR_KEYRING_BACKEND"

	keyringService = "rr"
	apiKeyItem     = "api_key"

	keyringTimeout = 30 * time.Second
)

// Store provides credential storage via system keyring.
type Store struct {
	ring keyring.Keyring
}

// NewStore creates a keyring-backed credential store.
func NewStore(backend string) (*Store, error) {
	if b := os.Getenv(envKeyringBackend); b != "" {
		backend = b
	}

	dir, err := config.EnsureKeyringDir()
	if err != nil {
		return nil, err
	}

	cfg := keyring.Config{
		ServiceName:              keyringService,
		KeychainTrustApplication: true,
		FileDir:                  dir,
		FilePasswordFunc:         fileKeyringPassword,
	}

	if backend != "" {
		cfg.AllowedBackends = []keyring.BackendType{keyring.BackendType(backend)}
	}

	ring, err := keyring.Open(cfg)
	if err != nil {
		return nil, fmt.Errorf("open keyring: %w", err)
	}

	return &Store{ring: ring}, nil
}

// SetAPIKey stores the API key in the keyring.
func (s *Store) SetAPIKey(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), keyringTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- s.ring.Set(keyring.Item{
			Key:  apiKeyItem,
			Data: []byte(key),
		})
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("store api key: %w", err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("store api key: %w", ctx.Err())
	}
}

// GetAPIKey returns the API key, checking env first then keyring.
func (s *Store) GetAPIKey() (string, error) {
	if key := os.Getenv(envAPIKey); key != "" {
		return key, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), keyringTimeout)
	defer cancel()

	type result struct {
		item keyring.Item
		err  error
	}

	done := make(chan result, 1)
	go func() {
		item, err := s.ring.Get(apiKeyItem)
		done <- result{item, err}
	}()

	select {
	case r := <-done:
		if r.err != nil {
			return "", fmt.Errorf("get api key: %w", r.err)
		}
		return string(r.item.Data), nil
	case <-ctx.Done():
		return "", fmt.Errorf("get api key: %w", ctx.Err())
	}
}

// DeleteAPIKey removes the API key from the keyring.
func (s *Store) DeleteAPIKey() error {
	ctx, cancel := context.WithTimeout(context.Background(), keyringTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- s.ring.Remove(apiKeyItem)
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("delete api key: %w", err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("delete api key: %w", ctx.Err())
	}
}

// HasAPIKey returns true if an API key is available (env or keyring).
func (s *Store) HasAPIKey() bool {
	if os.Getenv(envAPIKey) != "" {
		return true
	}
	key, err := s.GetAPIKey()
	return err == nil && key != ""
}

func fileKeyringPassword(_ string) (string, error) {
	if pw := os.Getenv(envKeyringPassword); pw != "" {
		return pw, nil
	}
	return "", fmt.Errorf("set %s for file-based keyring", envKeyringPassword)
}
