// Package cache provides caching functionality for jtk resources.
package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const Version = 1

var ErrCacheMiss = errors.New("cache miss")

// Envelope is the on-disk JSON shape for a single cached resource.
type Envelope[T any] struct {
	Resource  string    `json:"resource"`
	Instance  string    `json:"instance"`
	FetchedAt time.Time `json:"fetched_at"`
	TTL       string    `json:"ttl"`
	Version   int       `json:"version"`
	Data      T         `json:"data"`
}

// ReadResource reads the envelope for `name` from disk.
//   - Returns (envelope, nil) on success.
//   - Returns (zero, ErrCacheMiss) if the file does not exist.
//   - Returns (zero, error) on I/O or JSON decode failure.
//
// ReadResource does NOT check freshness; callers use freshness.go.
func ReadResource[T any](name string) (Envelope[T], error) {
	path, err := ResourceFile(name)
	if err != nil {
		return Envelope[T]{}, err
	}

	data, err := os.ReadFile(path) //nolint:gosec // path derives from config, not user input
	if err != nil {
		if os.IsNotExist(err) {
			return Envelope[T]{}, ErrCacheMiss
		}
		return Envelope[T]{}, fmt.Errorf("reading resource file: %w", err)
	}

	var env Envelope[T]
	if err := json.Unmarshal(data, &env); err != nil {
		return Envelope[T]{}, fmt.Errorf("parsing resource file: %w", err)
	}

	// A version mismatch is treated as a miss — the next write will overwrite
	// the envelope with the current schema. This makes schema bumps self-healing.
	if env.Version != Version {
		return Envelope[T]{}, ErrCacheMiss
	}

	return env, nil
}

// WriteResource atomically writes an envelope for `name`.
//   - TTL comes from the caller (registry entry). Resource, Instance, Version,
//     and FetchedAt are set by WriteResource itself.
//   - Atomic: write to a temp file in the same directory, then rename.
//   - Ensures the instance directory exists with mode 0700.
//   - Writes the file with mode 0600.
func WriteResource[T any](name string, ttl string, data T) error {
	instance, err := InstanceKey()
	if err != nil {
		return err
	}

	env := Envelope[T]{
		Resource:  name,
		Instance:  instance,
		FetchedAt: time.Now().UTC(),
		TTL:       ttl,
		Version:   Version,
		Data:      data,
	}
	return atomicWriteEnvelope(name, env)
}

// atomicWriteEnvelope marshals an envelope and writes it to the cache path for
// `name` using a temp-file-rename pattern. Shared by WriteResource (which
// constructs a fresh envelope) and writeRaw (which preserves existing metadata
// for Touch-style invalidation).
func atomicWriteEnvelope[T any](name string, env Envelope[T]) error {
	path, err := ResourceFile(name)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	jsonData, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling envelope: %w", err)
	}

	tmp, err := os.CreateTemp(dir, name+"-*.json.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(jsonData); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Chmod(tmpPath, 0o600); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("setting file mode: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("moving temp file to final path: %w", err)
	}

	return nil
}
