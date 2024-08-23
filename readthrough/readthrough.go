package readthrough

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func New(dir, prefix string) *ReadThrough {
	return &ReadThrough{dir: dir, prefix: prefix}
}

type ReadThrough struct {
	dir, prefix string
}

var ErrMiss = errors.New("cache miss")

func (rt *ReadThrough) Get(key string) (io.ReadCloser, string, error) {
	hash, filename := rt.hashAndFilename(key)

	if _, err := os.Stat(filename); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, hash, fmt.Errorf("error checking for cache file '%s': %w", hash, err)
	} else if err != nil {
		return nil, hash, fmt.Errorf("cache miss for '%s': %w", hash, ErrMiss)
	}

	cache, err := os.Open(filename)
	if err != nil {
		return nil, hash, fmt.Errorf("error opening cache file '%s' for read: %w", hash, err)
	}

	return cache, hash, nil
}

func (rt *ReadThrough) Set(key string, r io.ReadCloser) (io.ReadCloser, string, error) {
	hash, filename := rt.hashAndFilename(key)

	cache, err := os.Create(filename)
	if err != nil {
		return nil, hash, fmt.Errorf("error opening cache file '%s' for write: %w", hash, err)
	}
	defer cache.Close()

	var buf bytes.Buffer
	tee := io.TeeReader(r, cache)
	if _, err := io.Copy(&buf, tee); err != nil {
		return nil, hash, fmt.Errorf("error writing cache file '%s': %w", hash, err)
	}
	r.Close()

	return io.NopCloser(&buf), hash, nil
}

func (rt *ReadThrough) hashAndFilename(key string) (string, string) {
	var hasher = sha256.New()
	hasher.Write([]byte(key))
	hash := hex.EncodeToString(hasher.Sum(nil))
	return hash, filepath.Join(rt.dir, rt.prefix+hash)
}
