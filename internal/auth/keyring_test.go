package auth

import (
	"errors"
	"testing"

	"github.com/99designs/keyring"
)

type stubKeyring struct {
	setItem   keyring.Item
	setCalled bool
	setErr    error
}

func (s *stubKeyring) Get(string) (keyring.Item, error) {
	return keyring.Item{}, errors.New("not implemented")
}

func (s *stubKeyring) GetMetadata(string) (keyring.Metadata, error) {
	return keyring.Metadata{}, errors.New("not implemented")
}

//nolint:gocritic // keyring.Keyring requires keyring.Item by value.
func (s *stubKeyring) Set(item keyring.Item) error {
	s.setCalled = true
	s.setItem = item
	return s.setErr
}

func (s *stubKeyring) Remove(string) error {
	return errors.New("not implemented")
}

func (s *stubKeyring) Keys() ([]string, error) {
	return nil, errors.New("not implemented")
}

func TestStoreSetAPIKeyStoresLabeledItem(t *testing.T) {
	t.Parallel()

	store := &Store{ring: &stubKeyring{}}

	if err := store.SetAPIKey("secret-api-key"); err != nil {
		t.Fatalf("SetAPIKey returned error: %v", err)
	}

	ring := store.ring.(*stubKeyring)
	if !ring.setCalled {
		t.Fatal("expected keyring Set to be called")
	}
	if got, want := ring.setItem.Key, apiKeyItem; got != want {
		t.Fatalf("item key = %q, want %q", got, want)
	}
	if got, want := string(ring.setItem.Data), "secret-api-key"; got != want {
		t.Fatalf("item data = %q, want %q", got, want)
	}
	if got, want := ring.setItem.Label, "Realtime Register CLI API credential (api.yoursrs.com)"; got != want {
		t.Fatalf("item label = %q, want %q", got, want)
	}
}

func TestStoreSetAPIKeyPropagatesSetError(t *testing.T) {
	t.Parallel()

	store := &Store{ring: &stubKeyring{setErr: errors.New("boom")}}

	err := store.SetAPIKey("secret-api-key")
	if err == nil {
		t.Fatal("expected error")
	}
	if got, want := err.Error(), "store api key: boom"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}
