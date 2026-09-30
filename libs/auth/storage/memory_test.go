package storage

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"golang.org/x/oauth2"
)

func TestMemoryStore_PutAndLookup(t *testing.T) {
	s := NewMemoryStore()
	tok := &oauth2.Token{AccessToken: "abc", RefreshToken: "def"}
	if err := s.Put("key1", Entry{Token: tok}); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := s.Lookup("key1")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got.Token.AccessToken != "abc" {
		t.Errorf("AccessToken: want %q, got %q", "abc", got.Token.AccessToken)
	}
	if got.Token.RefreshToken != "def" {
		t.Errorf("RefreshToken: want %q, got %q", "def", got.Token.RefreshToken)
	}
}

func TestMemoryStore_PutAndLookupUseCopies(t *testing.T) {
	s := NewMemoryStore()
	tok := &oauth2.Token{AccessToken: "abc", RefreshToken: "def"}
	if err := s.Put("key1", Entry{Token: tok}); err != nil {
		t.Fatalf("Put: %v", err)
	}

	tok.RefreshToken = "mutated-after-store"

	got, err := s.Lookup("key1")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got.Token.RefreshToken != "def" {
		t.Fatalf("RefreshToken after store mutation: want %q, got %q", "def", got.Token.RefreshToken)
	}

	got.Token.RefreshToken = "mutated-after-lookup"

	gotAgain, err := s.Lookup("key1")
	if err != nil {
		t.Fatalf("Lookup after lookup mutation: %v", err)
	}
	if gotAgain.Token.RefreshToken != "def" {
		t.Fatalf("RefreshToken after lookup mutation: want %q, got %q", "def", gotAgain.Token.RefreshToken)
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	s := NewMemoryStore()
	if err := s.Put("key1", Entry{Token: &oauth2.Token{AccessToken: "abc"}}); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := s.Delete("key1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := s.Lookup("key1")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Lookup after Delete: want ErrNotFound, got %v", err)
	}
}

func TestMemoryStore_LookupUnsetKey(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.Lookup("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Lookup(missing): want ErrNotFound, got %v", err)
	}
}

func TestMemoryStore_ConcurrentPutAndLookup(t *testing.T) {
	s := NewMemoryStore()
	const goroutines = 20
	const iterations = 100
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)
	for i := range goroutines {
		prefix := fmt.Sprintf("writer-%d", i)
		go func() {
			defer wg.Done()
			for j := range iterations {
				key := fmt.Sprintf("%s-%d", prefix, j)
				_ = s.Put(key, Entry{Token: &oauth2.Token{AccessToken: key}})
			}
		}()
		go func() {
			defer wg.Done()
			for j := range iterations {
				_, _ = s.Lookup(fmt.Sprintf("%s-%d", prefix, j))
			}
		}()
	}
	wg.Wait()
}
