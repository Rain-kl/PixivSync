// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package oauth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
)

type cacheEntry struct {
	value     any
	expiredAt time.Time
}

type memoryCache struct {
	sync.RWMutex
	items map[string]cacheEntry
}

var localCache = &memoryCache{
	items: make(map[string]cacheEntry),
}

func (c *memoryCache) Set(key string, val any, ttl time.Duration) {
	c.Lock()
	defer c.Unlock()
	c.items[key] = cacheEntry{
		value:     val,
		expiredAt: time.Now().Add(ttl),
	}
}

func (c *memoryCache) Get(key string) (any, bool) {
	c.RLock()
	item, ok := c.items[key]
	if !ok {
		c.RUnlock()
		return nil, false
	}
	if time.Now().After(item.expiredAt) {
		c.RUnlock()
		c.Lock()
		if item, ok = c.items[key]; ok && time.Now().After(item.expiredAt) {
			delete(c.items, key)
		}
		c.Unlock()
		return nil, false
	}
	c.RUnlock()
	return item.value, true
}

func (c *memoryCache) Delete(key string) {
	c.Lock()
	defer c.Unlock()
	delete(c.items, key)
}

const (
	tokenCacheTTL = 5 * time.Minute
	userCacheTTL  = 5 * time.Minute
)

func tokenCacheKey(tokenHash string) string {
	return "oauth:token:" + tokenHash
}

func userCacheKey(userID uint64) string {
	return fmt.Sprintf("oauth:user:%d", userID)
}

// GetCachedToken 获取缓存的 AccessToken
func GetCachedToken(ctx context.Context, tokenHash string) (*model.AccessToken, error) {
	key := tokenCacheKey(tokenHash)
	if val, ok := localCache.Get(key); ok {
		if token, ok := val.(*model.AccessToken); ok {
			return token, nil
		}
	}

	if db.Redis != nil {
		var token model.AccessToken
		if err := db.GetJSON(ctx, key, &token); err == nil {
			// Write back to local cache
			localCache.Set(key, &token, tokenCacheTTL)
			return &token, nil
		}
	}
	return nil, fmt.Errorf("cache miss")
}

// SetCachedToken 设置 AccessToken 缓存
func SetCachedToken(ctx context.Context, tokenHash string, token *model.AccessToken) {
	key := tokenCacheKey(tokenHash)
	localCache.Set(key, token, tokenCacheTTL)
	if db.Redis != nil {
		_ = db.SetJSON(ctx, key, token, tokenCacheTTL)
	}
}

// InvalidateCachedToken 吊销/删除 token 缓存
func InvalidateCachedToken(ctx context.Context, tokenHash string) {
	key := tokenCacheKey(tokenHash)
	localCache.Delete(key)
	if db.Redis != nil {
		_ = db.Redis.Del(ctx, db.PrefixedKey(key)).Err()
	}
}

// GetCachedUser 获取缓存的 User
func GetCachedUser(ctx context.Context, userID uint64) (*model.User, error) {
	key := userCacheKey(userID)
	if val, ok := localCache.Get(key); ok {
		if u, ok := val.(*model.User); ok {
			return u, nil
		}
	}

	if db.Redis != nil {
		var u model.User
		if err := db.GetJSON(ctx, key, &u); err == nil {
			// Write back to local cache
			localCache.Set(key, &u, userCacheTTL)
			return &u, nil
		}
	}
	return nil, fmt.Errorf("cache miss")
}

// SetCachedUser 设置 User 缓存
func SetCachedUser(ctx context.Context, userID uint64, u *model.User) {
	key := userCacheKey(userID)
	localCache.Set(key, u, userCacheTTL)
	if db.Redis != nil {
		_ = db.SetJSON(ctx, key, u, userCacheTTL)
	}
}

// InvalidateCachedUser 吊销/失效 User 缓存
func InvalidateCachedUser(ctx context.Context, userID uint64) {
	key := userCacheKey(userID)
	localCache.Delete(key)
	if db.Redis != nil {
		_ = db.Redis.Del(ctx, db.PrefixedKey(key)).Err()
	}
}
