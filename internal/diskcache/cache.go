// Package diskcache implements a platform-level disk-backed cache with size limit, TTL, and LRU eviction.
package diskcache

import (
	"container/list"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/peterbourgon/diskv/v3"
)

// ErrCacheMiss represents a cache miss.
var ErrCacheMiss = errors.New("cache miss")

// Constants for disk cache configuration and sizing
const (
	defaultCacheDir        = "uploads/diskcache"
	headerSize             = 8 // 8 bytes metadata prefix for expiration UnixNano timestamp
	defaultMaxSizeMB       = 100
	defaultTTLMinutes      = 60
	defaultCleanupInterval = 10

	// DefaultExpiration applies the cache-wide default TTL.
	DefaultExpiration time.Duration = 0
	// NoExpiration stores the item without a TTL. Size limits and LRU eviction still apply.
	NoExpiration time.Duration = -1
)

var (
	globalCache     *DiskCache
	globalCacheOnce sync.Once
)

// Status represents the runtime cache statistics.
type Status struct {
	TotalSize  int64  `json:"total_size"`
	KeysCount  int    `json:"keys_count"`
	MaxSizeMB  int64  `json:"max_size_mb"`
	TTLMinutes int64  `json:"ttl_minutes"`
	LRUEnabled bool   `json:"lru_enabled"`
	BasePath   string `json:"base_path"`
}

// DiskCache implements the disk-backed cache with size limits, TTL, and LRU eviction.
type DiskCache struct {
	mu         sync.RWMutex
	d          *diskv.Diskv
	basePath   string
	maxSize    int64 // in bytes
	defaultTTL time.Duration
	lruEnabled bool

	// LRU and Size tracking
	currentSize int64
	items       map[string]*list.Element
	evictList   *list.List
}

type cacheItem struct {
	key       string
	size      int64
	expiredAt time.Time
}

// GetGlobalCache returns the global singleton DiskCache instance.
func GetGlobalCache() *DiskCache {
	globalCacheOnce.Do(func() {
		globalCache = New(defaultCacheDir)
		// Load initial configs from database
		globalCache.ReloadConfig(context.Background())
		// Start background routine to clean expired items every 10 minutes
		go globalCache.startCleanupWorker(defaultCleanupInterval * time.Minute)
	})
	return globalCache
}

// New creates a new DiskCache instance.
func New(basePath string) *DiskCache {
	d := diskv.New(diskv.Options{
		BasePath:     basePath,
		Transform:    func(_ string) []string { return []string{} }, // flat structure for easy walk
		CacheSizeMax: 1024 * 1024,                                   // 1MB in-memory cache size for diskv itself
	})

	c := &DiskCache{
		d:          d,
		basePath:   basePath,
		maxSize:    defaultMaxSizeMB * 1024 * 1024,  // 100MB default
		defaultTTL: defaultTTLMinutes * time.Minute, // 60 minutes default
		lruEnabled: true,
		items:      make(map[string]*list.Element),
		evictList:  list.New(),
	}

	// Scan directory on startup to rebuild LRU and size tracking
	_ = c.loadTracker()
	return c
}

// Set stores a key-value pair in the cache.
// Use DefaultExpiration for the configured default TTL, NoExpiration for no
// TTL, or a positive duration for a business-specific TTL.
func (c *DiskCache) Set(key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ttl == DefaultExpiration {
		ttl = c.defaultTTL
	}

	var expiredAt time.Time
	if ttl > 0 {
		expiredAt = time.Now().Add(ttl)
	}

	// Prepare data layout: 8 bytes expiration timestamp + raw payload
	buf := make([]byte, headerSize+len(value))
	var expNano int64
	if !expiredAt.IsZero() {
		expNano = expiredAt.UnixNano()
	}
	binary.BigEndian.PutUint64(buf[0:headerSize], uint64(expNano))
	copy(buf[headerSize:], value)

	// Write to diskv
	if err := c.d.Write(key, buf); err != nil {
		return fmt.Errorf("failed to write key to disk: %w", err)
	}

	// Get file size on disk (approximate)
	size := int64(len(buf))

	// Update memory tracker
	if elem, ok := c.items[key]; ok {
		item := elem.Value.(*cacheItem)
		c.currentSize += size - item.size
		item.size = size
		item.expiredAt = expiredAt
		c.evictList.MoveToFront(elem)
	} else {
		item := &cacheItem{
			key:       key,
			size:      size,
			expiredAt: expiredAt,
		}
		elem := c.evictList.PushFront(item)
		c.items[key] = elem
		c.currentSize += size
	}

	// Evict items if size limit exceeded and LRU is enabled
	c.evict()

	return nil
}

// Get retrieves a key's value from the cache.
func (c *DiskCache) Get(key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return nil, ErrCacheMiss
	}

	item := elem.Value.(*cacheItem)

	// Check expiration
	if !item.expiredAt.IsZero() && time.Now().After(item.expiredAt) {
		// Lazily delete expired item
		_ = c.deleteUnlocked(key)
		return nil, ErrCacheMiss
	}

	// Read from diskv
	data, err := c.d.Read(key)
	if err != nil {
		// Key exists in memory but not on disk, sync state
		_ = c.deleteUnlocked(key)
		return nil, ErrCacheMiss
	}

	if len(data) < headerSize {
		_ = c.deleteUnlocked(key)
		return nil, ErrCacheMiss
	}

	// Update LRU access order
	c.evictList.MoveToFront(elem)

	// Slice off the metadata header
	return data[headerSize:], nil
}

// Delete removes a key-value pair from the cache.
func (c *DiskCache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.deleteUnlocked(key)
}

func (c *DiskCache) deleteUnlocked(key string) error {
	if elem, ok := c.items[key]; ok {
		item := elem.Value.(*cacheItem)
		c.currentSize -= item.size
		c.evictList.Remove(elem)
		delete(c.items, key)
	}
	return c.d.Erase(key)
}

// Clear flushes all cached elements.
func (c *DiskCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.currentSize = 0
	c.items = make(map[string]*list.Element)
	c.evictList.Init()

	return c.d.EraseAll()
}

// Status returns the cache status.
func (c *DiskCache) Status() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return Status{
		TotalSize:  c.currentSize,
		KeysCount:  len(c.items),
		MaxSizeMB:  c.maxSize / (1024 * 1024),
		TTLMinutes: int64(c.defaultTTL.Minutes()),
		LRUEnabled: c.lruEnabled,
		BasePath:   c.basePath,
	}
}

// ReloadConfig reloads policies from database configs dynamically.
func (c *DiskCache) ReloadConfig(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Ensure DB is initialized before querying
	if db.DB(ctx) == nil {
		return
	}

	// 1. Max Size
	var scMaxSize model.SystemConfig
	maxSizeMB := int64(defaultMaxSizeMB)
	if err := scMaxSize.GetByKey(ctx, model.ConfigKeyDiskCacheMaxSizeMB); err == nil && scMaxSize.Value != "" {
		if val, err := strconv.ParseInt(scMaxSize.Value, 10, 64); err == nil && val > 0 {
			maxSizeMB = val
		}
	}
	c.maxSize = maxSizeMB * 1024 * 1024

	// 2. Default TTL
	var scTTL model.SystemConfig
	ttlMinutes := int64(defaultTTLMinutes)
	if err := scTTL.GetByKey(ctx, model.ConfigKeyDiskCacheTTLMinutes); err == nil && scTTL.Value != "" {
		if val, err := strconv.ParseInt(scTTL.Value, 10, 64); err == nil && val >= 0 {
			ttlMinutes = val
		}
	}
	c.defaultTTL = time.Duration(ttlMinutes) * time.Minute

	// 3. LRU Enabled
	var scLRU model.SystemConfig
	lruEnabled := true
	if err := scLRU.GetByKey(ctx, model.ConfigKeyDiskCacheLRUEnabled); err == nil && scLRU.Value != "" {
		if val, err := strconv.ParseBool(scLRU.Value); err == nil {
			lruEnabled = val
		}
	}
	c.lruEnabled = lruEnabled

	// Apply eviction immediately under new configs
	c.evict()
}

// evict evicts oldest items if current size exceeds maxSize and LRU is enabled.
func (c *DiskCache) evict() {
	if !c.lruEnabled {
		return
	}

	for c.currentSize > c.maxSize && c.evictList.Len() > 0 {
		elem := c.evictList.Back()
		if elem == nil {
			break
		}
		item := elem.Value.(*cacheItem)
		key := item.key

		// Remove from memory
		c.currentSize -= item.size
		c.evictList.Remove(elem)
		delete(c.items, key)

		// Delete from disk
		_ = c.d.Erase(key)
	}
}

// loadTracker walks the directory to rebuild the LRU and size tracking structures.
func (c *DiskCache) loadTracker() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.currentSize = 0
	c.items = make(map[string]*list.Element)
	c.evictList = list.New()

	files, err := os.ReadDir(c.basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	type tempItem struct {
		key       string
		size      int64
		expiredAt time.Time
		mtime     time.Time
	}
	var loadedItems []tempItem

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		// Skip temporary files
		if strings.HasPrefix(name, ".") || strings.Contains(name, "temp") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		filePath := filepath.Join(c.basePath, name)
		// #nosec G304
		f, err := os.Open(filePath)
		if err != nil {
			continue
		}

		var expiredAt time.Time
		var expNano int64
		err = binary.Read(f, binary.BigEndian, &expNano)
		_ = f.Close()
		if err != nil {
			// Corrupt metadata header: delete file
			_ = os.Remove(filePath)
			continue
		}

		if expNano > 0 {
			expiredAt = time.Unix(0, expNano)
			// Expired: delete file
			if time.Now().After(expiredAt) {
				_ = os.Remove(filePath)
				continue
			}
		}

		loadedItems = append(loadedItems, tempItem{
			key:       name,
			size:      info.Size(),
			expiredAt: expiredAt,
			mtime:     info.ModTime(),
		})
	}

	// Sort loaded items by modification time ascending (oldest first)
	sort.Slice(loadedItems, func(i, j int) bool {
		return loadedItems[i].mtime.Before(loadedItems[j].mtime)
	})

	// Populate LRU (PushFront so that newest items are at the front, oldest at the back)
	for _, item := range loadedItems {
		entry := &cacheItem{
			key:       item.key,
			size:      item.size,
			expiredAt: item.expiredAt,
		}
		element := c.evictList.PushFront(entry)
		c.items[item.key] = element
		c.currentSize += item.size
	}

	return nil
}

// startCleanupWorker periodically cleans up expired cache items.
func (c *DiskCache) startCleanupWorker(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		c.cleanExpired()
	}
}

// cleanExpired scans memory for expired items and removes them.
func (c *DiskCache) cleanExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, elem := range c.items {
		item := elem.Value.(*cacheItem)
		if !item.expiredAt.IsZero() && now.After(item.expiredAt) {
			c.currentSize -= item.size
			c.evictList.Remove(elem)
			delete(c.items, key)
			_ = c.d.Erase(key)
		}
	}
}
