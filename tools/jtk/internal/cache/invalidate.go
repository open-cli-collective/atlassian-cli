package cache

import (
	"encoding/json"
	"errors"
	"time"
)

// Touch marks one or more caches stale by zeroing their FetchedAt timestamp.
// Missing envelope files (ErrCacheMiss) are silently skipped. Returns the first
// non-miss I/O error encountered.
func Touch(names ...string) error {
	for _, name := range names {
		env, err := ReadResource[json.RawMessage](name)
		if errors.Is(err, ErrCacheMiss) {
			continue
		}
		if err != nil {
			return err
		}

		// Zero out FetchedAt to mark stale, but preserve the existing Data bytes.
		env.FetchedAt = time.Time{}
		if err := writeRaw(name, env); err != nil {
			return err
		}
	}
	return nil
}

// AppendOnCreate appends `item` to the Data slice of `name`'s envelope and
// writes it back atomically. Missing envelope files are silently skipped.
// T must match the element type of the envelope's Data slice.
func AppendOnCreate[T any](name string, item T) error {
	env, err := ReadResource[[]T](name)
	if errors.Is(err, ErrCacheMiss) {
		return nil
	}
	if err != nil {
		return err
	}

	env.Data = append(env.Data, item)
	return WriteResource(name, env.TTL, env.Data)
}

// RemoveOnDelete removes items from the Data slice of `name`'s envelope for
// which `match` returns true, and writes the result back atomically. Missing
// envelope files are silently skipped. Does nothing if no items match.
func RemoveOnDelete[T any](name string, match func(T) bool) error {
	env, err := ReadResource[[]T](name)
	if errors.Is(err, ErrCacheMiss) {
		return nil
	}
	if err != nil {
		return err
	}

	// Filter: keep items where match returns false.
	filtered := make([]T, 0, len(env.Data))
	for _, item := range env.Data {
		if !match(item) {
			filtered = append(filtered, item)
		}
	}

	// No change, don't write.
	if len(filtered) == len(env.Data) {
		return nil
	}

	return WriteResource(name, env.TTL, filtered)
}

// writeRaw atomically writes an envelope while preserving its FetchedAt field
// (Touch zeros it to mark stale; WriteResource would set a fresh timestamp).
// Delegates to envelope.go's shared atomicWriteEnvelope helper.
func writeRaw(name string, env Envelope[json.RawMessage]) error {
	return atomicWriteEnvelope(name, env)
}
