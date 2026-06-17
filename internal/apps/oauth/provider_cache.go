// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package oauth

import (
	"context"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/sync/singleflight"
)

// oidcProviderCache 进程级 OIDC provider 缓存。
//
// oidc.NewProvider 每次调用都会向远端 issuer 的
// /.well-known/openid-configuration 发起 HTTP 请求拉取元数据。
// 由于 provider 元数据极少变动，将其缓存后可消除登录发起与回调时的
// 重复外部 HTTP 往返。
//
// 并发安全性：
//   - mu + entries 防止并发读写 map。
//   - sfGroup 保证同一 issuer 同时只有一次在途的 NewProvider 调用
//     （singleflight），后续等待者复用同一结果，彻底消除 thundering herd。
type oidcProviderCache struct {
	mu      sync.RWMutex
	entries map[string]*oidc.Provider // key: normalized issuer URL
	sfGroup singleflight.Group
}

// globalOIDCProviderCache 是包级单例缓存，与进程同生命周期。
var globalOIDCProviderCache = &oidcProviderCache{
	entries: make(map[string]*oidc.Provider),
}

// get 返回缓存的 provider；若无则通过 oidc.NewProvider 获取并写入缓存。
// 同一 issuer 并发调用时，singleflight 保证只有一次实际 HTTP 请求。
//
// 注意：接受 _ 形参以与调用方类型一致，内部有意使用 context.Background() 而非传入的请求 ctx，
// 以防止请求被提前取消时导致缓存写入失败。
func (c *oidcProviderCache) get(_ context.Context, issuer string) (*oidc.Provider, error) { //nolint:contextcheck // intentional: use Background to avoid request cancellation affecting cache write
	// 快路径：已有缓存则直接返回。
	c.mu.RLock()
	if p, ok := c.entries[issuer]; ok {
		c.mu.RUnlock()
		return p, nil
	}
	c.mu.RUnlock()

	// 慢路径：通过 singleflight 合并并发的首次请求。
	// 闭包内有意使用 context.Background() 而非请求 ctx，防止请求取消导致缓存写入失败。
	v, err, _ := c.sfGroup.Do(issuer, func() (any, error) { //nolint:contextcheck // intentional: Background ctx prevents cache write failure on request cancellation
		// 双检：singleflight 内再次检查，前一个并发组可能已写入缓存。
		c.mu.RLock()
		if p, ok := c.entries[issuer]; ok {
			c.mu.RUnlock()
			return p, nil
		}
		c.mu.RUnlock()

		p, err := oidc.NewProvider(context.Background(), issuer)
		if err != nil {
			return nil, err
		}

		c.mu.Lock()
		c.entries[issuer] = p
		c.mu.Unlock()
		return p, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*oidc.Provider), nil //nolint:forcetypeassert // singleflight value 由同函数写入，类型确定
}

// invalidate 从缓存中移除指定 issuer 对应的 provider。
// 在认证源的 Discovery URL 被修改时调用，强制下次请求重新拉取元数据。
func (c *oidcProviderCache) invalidate(issuer string) {
	c.mu.Lock()
	delete(c.entries, issuer)
	c.mu.Unlock()
}

// InvalidateOIDCProviderCache 从进程级缓存中清除指定 issuer 的 provider 条目。
// 当管理员更新认证源的 Discovery URL 后调用，以确保下次登录时重新拉取最新元数据。
// issuer 值应为去掉 /.well-known/openid-configuration 后缀的规范化 URL。
func InvalidateOIDCProviderCache(issuer string) {
	globalOIDCProviderCache.invalidate(issuer)
}
