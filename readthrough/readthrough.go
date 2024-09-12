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
	hash, dirname, filename := rt.hashAndFilename(key)
	path := filepath.Join(dirname, filename)

	if _, err := os.Stat(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, hash, fmt.Errorf("error checking for cache file '%s': %w", hash, err)
	} else if err != nil {
		return nil, hash, fmt.Errorf("cache miss for '%s': %w", hash, ErrMiss)
	}

	cache, err := os.Open(path)
	if err != nil {
		return nil, hash, fmt.Errorf("error opening cache file '%s' for read: %w", path, err)
	}

	return cache, hash, nil
}

func (rt *ReadThrough) Set(key string, r io.ReadCloser) (io.ReadCloser, string, error) {
	hash, dirname, filename := rt.hashAndFilename(key)

	if err := os.MkdirAll(dirname, 0755); err != nil {
		return nil, hash, fmt.Errorf("error creating cache dir '%s': %w", dirname, err)
	}
	path := filepath.Join(dirname, filename)

	cache, err := os.Create(path)
	if err != nil {
		return nil, hash, fmt.Errorf("error opening cache file '%s' for write: %w", path, err)
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

func (rt *ReadThrough) hashAndFilename(key string) (string, string, string) {
	var hasher = sha256.New()
	hasher.Write([]byte(key))
	hash := hex.EncodeToString(hasher.Sum(nil))
	first, second, third := hash[:2], hash[2:4], hash[4:]
	return hash, filepath.Join(rt.dir, rt.prefix+first, second), third
}
