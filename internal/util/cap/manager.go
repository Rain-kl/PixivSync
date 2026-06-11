// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package cap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rain-kl/Wavelet/internal/config"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
)

const (
	managerDefaultChallengeCount = 1
	managerDefaultChallengeSize  = 32
	defaultChallengeDifficulty   = 4
	defaultChallengeTTL          = 10 * time.Minute
	defaultTokenTTL              = 20 * time.Minute
	redeemTokenIDLength          = 8  // 兑换 Token ID 字节长度
	redeemVerTokenLength         = 15 // 兑换验证 Token 字节长度
	tokenPartsCount              = 2  // 兑换 Token 由两部分组成
	valuePartsCount              = 2  // 存储值由 scope 和过期时间组成
)

// Config holds settings for the CAPTCHA manager
type Config struct {
	Secret              []byte        // HMAC signing key
	ChallengeCount      int           // Number of PoW puzzles
	ChallengeSize       int           // Size of the salt string
	ChallengeDifficulty int           // Length of difficulty target prefix
	ChallengeTTL        time.Duration // Lifespan of the challenge JWT
	TokenTTL            time.Duration // Lifespan of the redeem token
}

// Manager orchestrates challenge generation and solution validation
type Manager struct {
	conf  Config
	store Store
}

// NewManager creates a new CAPTCHA Manager
func NewManager(conf Config, store Store) *Manager {
	if conf.ChallengeCount <= 0 {
		conf.ChallengeCount = managerDefaultChallengeCount
	}
	if conf.ChallengeSize <= 0 {
		conf.ChallengeSize = managerDefaultChallengeSize
	}
	if conf.ChallengeDifficulty <= 0 {
		conf.ChallengeDifficulty = defaultChallengeDifficulty
	}
	if conf.ChallengeTTL <= 0 {
		conf.ChallengeTTL = defaultChallengeTTL
	}
	if conf.TokenTTL <= 0 {
		conf.TokenTTL = defaultTokenTTL
	}
	return &Manager{
		conf:  conf,
		store: store,
	}
}

// Generate creates a challenge response
func (m *Manager) Generate(ctx context.Context, scope string) (*ChallengeResponse, error) {
	c := ChallengeConfig{
		Count:      m.getChallengeCount(ctx),
		Size:       m.getChallengeSize(ctx),
		Difficulty: m.getChallengeDifficulty(ctx),
		Expires:    m.getChallengeTTL(ctx),
	}
	return GenerateChallenge(m.conf.Secret, c, scope)
}

// Redeem verifies PoW solutions and returns a one-time redeem token
func (m *Manager) Redeem(ctx context.Context, token string, solutions []int, scope string) (*RedeemResponse, error) {
	sigHex := jwtSigHex(token)
	if sigHex == "" {
		return &RedeemResponse{Success: false, Error: "invalid_token"}, nil
	}

	nonceKey := "cap:nonce:" + sigHex

	// Atomically claim the nonce slot BEFORE verifying solutions.
	// SetNX returns true only when the key did not previously exist, so two
	// concurrent requests carrying the same JWT can never both succeed here.
	// TTL is set to the challenge's remaining lifetime so the slot auto-expires.
	payload, err := VerifyChallengeSolutions(token, solutions, m.conf.Secret, scope)
	if err != nil {
		return &RedeemResponse{Success: false, Error: err.Error()}, nil //nolint:nilerr // expected behavior: validation error is returned as response, not system error
	}

	// Calculate remaining lifetime of the challenge JWT for the nonce TTL.
	now := time.Now().UnixNano() / int64(time.Millisecond)
	nonceTTL := time.Duration(payload.Expires-now) * time.Millisecond
	if nonceTTL < time.Second {
		nonceTTL = time.Second
	}

	// Atomic claim: if another goroutine already redeemed this JWT the SetNX
	// will return false and we reject the request without issuing a token.
	set, err := m.store.SetNX(ctx, nonceKey, "1", nonceTTL)
	if err != nil {
		return &RedeemResponse{Success: false, Error: "nonce_store_error"}, err
	}
	if !set {
		return &RedeemResponse{Success: false, Error: "already_redeemed"}, nil
	}

	// Generate a redeem token formatted as "id:verToken"
	id := randomHex(redeemTokenIDLength)
	verToken := randomHex(redeemVerTokenLength)
	verHashBytes := sha256.Sum256([]byte(verToken))
	verHashHex := hex.EncodeToString(verHashBytes[:])

	tokenKey := "cap:token:" + id + ":" + verHashHex
	tokenTTL := m.getTokenTTL(ctx)
	tokenExpires := time.Now().Add(tokenTTL)

	// Value stored is "expiresNano|scope"
	storeVal := strconv.FormatInt(tokenExpires.UnixNano(), 10) + "|" + scope

	if err := m.store.Set(ctx, tokenKey, storeVal, tokenTTL); err != nil {
		return &RedeemResponse{Success: false, Error: "token_store_error"}, err
	}

	return &RedeemResponse{
		Success: true,
		Token:   id + ":" + verToken,
		Expires: tokenExpires.UnixNano() / int64(time.Millisecond),
	}, nil
}

// VerifyToken validates and consumes the redeem token (single-use).
// GetAndDelete is used so that retrieval and removal happen atomically:
// two concurrent requests carrying the same token can never both see a value.
func (m *Manager) VerifyToken(ctx context.Context, token string, expectedScope string) (bool, error) {
	if token == "" {
		return false, nil
	}
	parts := strings.Split(token, ":")
	if len(parts) != tokenPartsCount {
		return false, nil
	}
	id := parts[0]
	verToken := parts[1]

	verHashBytes := sha256.Sum256([]byte(verToken))
	verHashHex := hex.EncodeToString(verHashBytes[:])

	tokenKey := "cap:token:" + id + ":" + verHashHex

	// Atomically retrieve-and-delete: the first caller gets the value, any
	// subsequent caller (even concurrent) receives (false, nil) immediately.
	val, exists, err := sGetAndDelete(ctx, m.store, tokenKey)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	valParts := strings.Split(val, "|")
	if len(valParts) != valuePartsCount {
		return false, nil
	}

	expNano, err := strconv.ParseInt(valParts[0], 10, 64)
	if err != nil {
		return false, nil //nolint:nilerr // expected behavior: invalid format is treated as validation failure, not system error
	}
	tokenScope := valParts[1]

	if expectedScope != "" && tokenScope != expectedScope {
		return false, nil
	}

	if time.Now().UnixNano() > expNano {
		return false, nil // Expired
	}

	return true, nil
}

// sGetAndDelete safely calls store.GetAndDelete, treating a nil store as a miss.
func sGetAndDelete(ctx context.Context, store Store, key string) (string, bool, error) {
	if store == nil {
		return "", false, nil
	}
	return store.GetAndDelete(ctx, key)
}

func (m *Manager) getChallengeCount(ctx context.Context) int {
	val, err := model.GetIntByKey(ctx, model.ConfigKeyCapChallengeCount)
	if err != nil || val <= 0 {
		return m.conf.ChallengeCount
	}
	return val
}

func (m *Manager) getChallengeSize(ctx context.Context) int {
	val, err := model.GetIntByKey(ctx, model.ConfigKeyCapChallengeSize)
	if err != nil || val <= 0 {
		return m.conf.ChallengeSize
	}
	return val
}

func (m *Manager) getChallengeDifficulty(ctx context.Context) int {
	val, err := model.GetIntByKey(ctx, model.ConfigKeyCapChallengeDifficulty)
	if err != nil || val <= 0 {
		return m.conf.ChallengeDifficulty
	}
	return val
}

func (m *Manager) getChallengeTTL(ctx context.Context) time.Duration {
	val, err := model.GetIntByKey(ctx, model.ConfigKeyCapChallengeTTL)
	if err != nil || val <= 0 {
		return m.conf.ChallengeTTL
	}
	return time.Duration(val) * time.Second
}

func (m *Manager) getTokenTTL(ctx context.Context) time.Duration {
	val, err := model.GetIntByKey(ctx, model.ConfigKeyCapTokenTTL)
	if err != nil || val <= 0 {
		return m.conf.TokenTTL
	}
	return time.Duration(val) * time.Second
}

var (
	defaultManager *Manager
	once           sync.Once
)

// GetDefaultManager yields the global singleton CAPTCHA manager
func GetDefaultManager() *Manager {
	once.Do(func() {
		var secret []byte
		if config.Config != nil && config.Config.App.SessionSecret != "" {
			secret = []byte(config.Config.App.SessionSecret)
		} else {
			secret = []byte("default-captcha-secret-key-at-least-16-bytes")
		}

		challengeCount := managerDefaultChallengeCount
		challengeSize := managerDefaultChallengeSize
		challengeDifficulty := defaultChallengeDifficulty
		challengeTTL := defaultChallengeTTL
		tokenTTL := defaultTokenTTL

		var store Store
		if config.Config != nil && config.Config.Redis.Enabled && db.Redis != nil {
			store = NewRedisStore(db.Redis)
		} else {
			store = NewMemoryStore(1 * time.Minute)
		}

		defaultManager = NewManager(Config{
			Secret:              secret,
			ChallengeCount:      challengeCount,
			ChallengeSize:       challengeSize,
			ChallengeDifficulty: challengeDifficulty,
			ChallengeTTL:        challengeTTL,
			TokenTTL:            tokenTTL,
		}, store)
	})
	return defaultManager
}
