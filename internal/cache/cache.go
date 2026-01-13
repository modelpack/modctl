/*
 *     Copyright 2025 The CNAI Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cache

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

const (
	// TTL is the time-to-live for cached items.
	TTL = 24 * time.Hour

	// FileLockRetryDelay is the delay between retries when acquiring file locks.
	FileLockRetryDelay = 100 * time.Millisecond
)

// ErrNotFound is returned when an item is not found in the cache.
var ErrNotFound = errors.New("item not found")

// Cache is the interface for caching file related information.
type Cache interface {
	// Get retrieves an item from the cache.
	Get(ctx context.Context, path string) (*Item, error)

	// Put inserts or updates an item in the cache.
	Put(ctx context.Context, item *Item) error
}

// Item represents a cached file item.
type Item struct {
	// Path is the absolute path of the file.
	Path string `json:"path"`

	// ModTime is the last modification time of the file.
	ModTime time.Time `json:"mod_time"`

	// Size is the size of the file in bytes.
	Size int64 `json:"size"`

	// Digest is the SHA-256 digest of the file.
	Digest string `json:"digest"`

	// CreatedAt is the time when the item was created.
	CreatedAt time.Time `json:"created_at"`
}

// cache is the implementation of the Cache interface.
type cache struct {
	// storageDir is the directory where the cache items are stored.
	storageDir string

	// flock is the file lock for the cache file.
	flock *flock.Flock
}

// New creates a new cache instance.
func New(storageDir string) (Cache, error) {
	c := &cache{
		storageDir: storageDir,
	}

	// Ensure cache directory exists.
	cacheDir := filepath.Dir(c.storagePath())
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, err
	}

	c.flock = flock.New(c.storagePath())
	return c, nil
}

// storagePath returns the path to the storage cache file.
func (c *cache) storagePath() string {
	return filepath.Join(c.storageDir, "modctl-cache.json")
}

// readItems reads all items from the cache file without locking.
// The caller must hold the lock.
func (c *cache) readItems() (map[string]*Item, error) {
	data, err := os.ReadFile(c.storagePath())
	if err != nil {
		// If the file doesn't exist, return an empty map.
		if os.IsNotExist(err) {
			return make(map[string]*Item), nil
		}
		return nil, err
	}

	// Handle empty file.
	if len(data) == 0 {
		return make(map[string]*Item), nil
	}

	var items []*Item
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}

	itemMap := make(map[string]*Item, len(items))
	for _, item := range items {
		itemMap[item.Path] = item
	}

	return itemMap, nil
}

// writeItems writes items to the cache file without locking.
// The caller must hold the lock.
func (c *cache) writeItems(itemsMap map[string]*Item) error {
	items := make([]*Item, 0, len(itemsMap))
	for _, item := range itemsMap {
		items = append(items, item)
	}

	data, err := json.Marshal(items)
	if err != nil {
		return err
	}

	return os.WriteFile(c.storagePath(), data, 0644)
}

// prune removes expired items from the map in-place.
func (c *cache) prune(itemsMap map[string]*Item) {
	now := time.Now()
	for path, item := range itemsMap {
		if now.Sub(item.CreatedAt) > TTL {
			delete(itemsMap, path)
		}
	}
}

// Get retrieves an item from the cache.
func (c *cache) Get(ctx context.Context, path string) (*Item, error) {
	// Check context before locking
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if _, err := c.flock.TryLockContext(ctx, FileLockRetryDelay); err != nil {
		return nil, err
	}
	defer c.flock.Unlock()

	items, err := c.readItems()
	if err != nil {
		return nil, err
	}

	item, ok := items[path]
	if !ok {
		return nil, ErrNotFound
	}

	// If the item is expired, return not found.
	if time.Since(item.CreatedAt) > TTL {
		return nil, ErrNotFound
	}

	return item, nil
}

// Put inserts or updates an item in the cache.
func (c *cache) Put(ctx context.Context, item *Item) error {
	// Check context before locking.
	if err := ctx.Err(); err != nil {
		return err
	}

	if _, err := c.flock.TryLockContext(ctx, FileLockRetryDelay); err != nil {
		return err
	}
	defer c.flock.Unlock()

	// Read existing items.
	itemsMap, err := c.readItems()
	if err != nil {
		return err
	}

	// Update or insert the item.
	itemsMap[item.Path] = item

	// Prune expired items.
	c.prune(itemsMap)

	// Write back to file.
	return c.writeItems(itemsMap)
}
