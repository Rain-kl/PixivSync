// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package oauth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/go-jose/go-jose/v4"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Rain-kl/Wavelet/internal/config"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/util"
)

// -----------------------------------------------------------------------------
// Mocks Setup
// -----------------------------------------------------------------------------

type mockRedisClient struct {
	redis.UniversalClient
	store map[string]string
}

func newMockRedisClient() *mockRedisClient {
	return &mockRedisClient{
		store: make(map[string]string),
	}
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	var val string
	switch v := value.(type) {
	case []byte:
		val = string(v)
	case string:
		val = v
	default:
		val = fmt.Sprintf("%v", v)
	}
	m.store[key] = val
	cmd.SetVal("OK")
	return cmd
}

func (m *mockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)
	val, ok := m.store[key]
	if !ok {
		cmd.SetErr(redis.Nil)
	} else {
		cmd.SetVal(val)
	}
	return cmd
}

func (m *mockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	var count int64
	for _, key := range keys {
		if _, ok := m.store[key]; ok {
			delete(m.store, key)
			count++
		}
	}
	cmd.SetVal(count)
	return cmd
}

func (m *mockRedisClient) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	if len(values) >= 2 {
		field := fmt.Sprintf("%v", values[0])
		var val string
		switch v := values[1].(type) {
		case []byte:
			val = string(v)
		case string:
			val = v
		default:
			val = fmt.Sprintf("%v", v)
		}
		compositeKey := key + ":" + field
		m.store[compositeKey] = val
		cmd.SetVal(1)
	} else {
		cmd.SetVal(0)
	}
	return cmd
}

func (m *mockRedisClient) HGet(ctx context.Context, key string, field string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)
	compositeKey := key + ":" + field
	val, ok := m.store[compositeKey]
	if !ok {
		cmd.SetErr(redis.Nil)
	} else {
		cmd.SetVal(val)
	}
	return cmd
}

type mockRoundTripper struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

// Global cryptographic tools for custom OIDC mocking
var (
	testRSAPrivateKey *rsa.PrivateKey
	testJWKS          jose.JSONWebKeySet
)

const (
	testIssuerURL     = "https://connect.linux.do"
	testAuthURL       = "https://connect.linux.do/oauth2/authorize"
	testTokenURL      = "https://connect.linux.do/oauth2/token"
	testJWKSURL       = "https://connect.linux.do/oauth2/keys"
	testClientID      = "test_client_id"
	testClientSecret  = "test_client_secret"
	testSourceName    = "linuxdo"
	testSourceDisplay = "LINUX DO"
)

func init() {
	var err error
	testRSAPrivateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(fmt.Sprintf("failed to generate RSA key: %v", err))
	}
	jwk := jose.JSONWebKey{
		Key:       &testRSAPrivateKey.PublicKey,
		KeyID:     "test-key-id",
		Algorithm: string(jose.RS256),
		Use:       "sig",
	}
	testJWKS = jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{jwk},
	}
}

func seedTestAuthSource(t *testing.T, dbConn *gorm.DB) {
	t.Helper()
	if err := dbConn.Create(&model.AuthSource{
		ID:                 100,
		Name:               testSourceName,
		Type:               model.AuthSourceTypeOIDC,
		DisplayName:        testSourceDisplay,
		IsActive:           true,
		ClientID:           testClientID,
		ClientSecret:       testClientSecret,
		OpenIDDiscoveryURL: testIssuerURL,
	}).Error; err != nil {
		t.Fatalf("failed to seed auth source: %v", err)
	}
}

func oidcDiscoveryResponse() *http.Response {
	body := fmt.Sprintf(`{
		"issuer": %q,
		"authorization_endpoint": %q,
		"token_endpoint": %q,
		"jwks_uri": %q,
		"response_types_supported": ["code"],
		"subject_types_supported": ["public"],
		"id_token_signing_alg_values_supported": ["RS256"]
	}`, testIssuerURL, testAuthURL, testTokenURL, testJWKSURL)
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

type mockClaims struct {
	ID       uint64 `json:"id"`
	Issuer   string `json:"iss"`
	Subject  string `json:"sub"`
	Audience string `json:"aud"`
	Expiry   int64  `json:"exp"`
	IssuedAt int64  `json:"iat"`
	Nonce    string `json:"nonce"`
	Username string `json:"preferred_username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Active   bool   `json:"active"`
}

func generateMockIDToken(issuer, sub, aud, nonce, username, email, name string) string {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: testRSAPrivateKey}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		panic(err)
	}
	id, _ := strconv.ParseUint(sub, 10, 64)
	claims := mockClaims{
		ID:       id,
		Issuer:   issuer,
		Subject:  sub,
		Audience: aud,
		Expiry:   time.Now().Add(time.Hour).Unix(),
		IssuedAt: time.Now().Unix(),
		Nonce:    nonce,
		Username: username,
		Email:    email,
		Name:     name,
		Active:   true,
	}
	payload, _ := json.Marshal(claims)
	object, err := signer.Sign(payload)
	if err != nil {
		panic(err)
	}
	tokenStr, _ := object.CompactSerialize()
	return tokenStr
}

// -----------------------------------------------------------------------------
// Test Helpers
func newMockOIDCClient(issuer, clientID, expectedState, sub, username, email, name string) *http.Client {
	cleanIssuer := strings.TrimRight(issuer, "/")
	return &http.Client{
		Transport: &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				urlStr := req.URL.String()
				if req.Method == http.MethodGet && strings.Contains(urlStr, "/.well-known/openid-configuration") {
					body := fmt.Sprintf(`{
						"issuer": %q,
						"authorization_endpoint": %q,
						"token_endpoint": %q,
						"jwks_uri": %q,
						"response_types_supported": ["code"],
						"subject_types_supported": ["public"],
						"id_token_signing_alg_values_supported": ["RS256"]
					}`, cleanIssuer, cleanIssuer+"/oauth2/authorize", cleanIssuer+"/oauth2/token", cleanIssuer+"/oauth2/keys")
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(body)),
						Header:     make(http.Header),
					}, nil
				}
				if req.Method == http.MethodGet && (strings.Contains(urlStr, "/keys") || strings.Contains(urlStr, "/jwks")) {
					jwksJSON, _ := json.Marshal(testJWKS)
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(jwksJSON)),
						Header:     make(http.Header),
					}, nil
				}
				if req.Method == http.MethodPost && (strings.Contains(urlStr, "/token") || strings.Contains(urlStr, "/access_token")) {
					idToken := generateMockIDToken(issuer, sub, clientID, expectedState, username, email, name)
					body := fmt.Sprintf(`{"access_token":"mock_access_token","token_type":"Bearer","expires_in":3600,"id_token":"%s"}`, idToken)
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(body)),
						Header:     make(http.Header),
					}, nil
				}
				return nil, fmt.Errorf("unexpected mock request: %s %s", req.Method, req.URL)
			},
		},
	}
}

func setupTestDB(t *testing.T) *gorm.DB {
	dbConn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite in memory: %v", err)
	}
	err = dbConn.AutoMigrate(
		&model.User{},
		&model.AuthSource{},
		&model.ExternalAccount{},
		&model.SystemConfig{},
	)
	if err != nil {
		t.Fatalf("failed to migrate schema: %v", err)
	}

	// 注入测试所需的服务器地址配置
	if err := dbConn.Create(&model.SystemConfig{
		Key:   model.ConfigKeyServerAddress,
		Value: "http://localhost:3000",
	}).Error; err != nil {
		t.Fatalf("failed to seed server_address config: %v", err)
	}

	return dbConn
}

func mockContextMiddleware(mockClient *http.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, oauth2.HTTPClient, mockClient)
		ctx = oidc.ClientContext(ctx, mockClient)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func setupTestRouter(dbConn *gorm.DB, mockRedis *mockRedisClient, mockClient *http.Client) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Inject context mock middleware
	r.Use(mockContextMiddleware(mockClient))

	store := cookie.NewStore([]byte(config.Config.App.SessionSecret))
	store.Options(util.GetSessionOptions(3600))
	r.Use(sessions.Sessions(config.Config.App.SessionCookieName, store))

	db.SetDB(dbConn)
	db.Redis = mockRedis

	api := r.Group("/api/v1")
	{
		api.GET("/oauth/sources", GetLoginSources)
		api.GET("/oauth/login", GetLoginURL)
		api.GET("/oauth/:source/authorize", Authorize)
		api.GET("/oauth/logout", Logout)
		api.POST("/oauth/callback", Callback)
		api.GET("/oauth/user-info", LoginRequired(), UserInfo)
		api.GET("/oauth/external-accounts", LoginRequired(), ListExternalAccounts)
		api.POST("/oauth/external-accounts/:id/delete", LoginRequired(), DeleteExternalAccount)
	}

	return r
}

func performRequest(r http.Handler, method, path string, body []byte, headers map[string]string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, _ := http.NewRequest(method, path, bodyReader)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	for _, cookie := range cookies {
		if cookie != nil {
			req.AddCookie(cookie)
		}
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func initializeTestConfig() {
	config.Config.App.Env = "testing"
	config.Config.App.SessionCookieName = "test_session_id"
	config.Config.App.SessionSecret = "test_session_secret"
	config.Config.App.APIPrefix = "/api"
}

// -----------------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------------

func TestGetLoginSources(t *testing.T) {
	initializeTestConfig()
	dbConn := setupTestDB(t)
	mockRedis := newMockRedisClient()

	// Setup empty HTTP Mock
	httpMock := &http.Client{
		Transport: &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("unexpected request")
			},
		},
	}
	util.SetHTTPClient(httpMock)
	router := setupTestRouter(dbConn, mockRedis, httpMock)

	// Inject OIDC login enabled config
	dbConn.Create(&model.SystemConfig{
		Key:   model.ConfigKeyOIDCLoginEnabled,
		Value: "true",
	})

	// Inject active DB auth source
	dbConn.Create(&model.AuthSource{
		ID:                 101,
		Name:               "github",
		Type:               model.AuthSourceTypeOIDC,
		DisplayName:        "GitHub OAuth",
		IsActive:           true,
		ClientID:           "gh_client",
		ClientSecret:       "gh_secret",
		OpenIDDiscoveryURL: "https://github.com",
	})

	// Perform GET request
	w := performRequest(router, http.MethodGet, "/api/v1/oauth/sources", nil, nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Data []AuthSourceView `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 active source, got %d", len(resp.Data))
	}

	if resp.Data[0].Name != "github" {
		t.Errorf("expected github source, got %s", resp.Data[0].Name)
	}

	// Test disabling OIDC
	dbConn.Model(&model.SystemConfig{}).Where("key = ?", model.ConfigKeyOIDCLoginEnabled).Update("value", "false")
	mockRedis.store = make(map[string]string)

	w2 := performRequest(router, http.MethodGet, "/api/v1/oauth/sources", nil, nil, nil)
	var resp2 struct {
		Data []AuthSourceView `json:"data"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)
	if len(resp2.Data) != 0 {
		t.Errorf("expected 0 sources when OIDC is disabled, got %d", len(resp2.Data))
	}
}

func TestGetLoginURL(t *testing.T) {
	initializeTestConfig()
	dbConn := setupTestDB(t)
	mockRedis := newMockRedisClient()
	seedTestAuthSource(t, dbConn)

	httpMock := &http.Client{
		Transport: &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet && strings.Contains(req.URL.String(), "/.well-known/openid-configuration") {
					return oidcDiscoveryResponse(), nil
				}
				return nil, fmt.Errorf("unexpected request")
			},
		},
	}
	util.SetHTTPClient(httpMock)
	router := setupTestRouter(dbConn, mockRedis, httpMock)

	// Case 1: Default Login URL
	w := performRequest(router, http.MethodGet, "/api/v1/oauth/login", nil, nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Data OAuthAuthorizeResponse `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !strings.Contains(resp.Data.AuthorizeURL, testAuthURL) {
		t.Errorf("invalid authorize URL: %s", resp.Data.AuthorizeURL)
	}

	parsedURL, _ := url.Parse(resp.Data.AuthorizeURL)
	state := parsedURL.Query().Get("state")
	if state == "" {
		t.Error("missing state in URL")
	}

	redisKey := db.PrefixedKey(fmt.Sprintf(OAuthStateCacheKeyFormat, state))
	stateVal, err := mockRedis.Get(context.Background(), redisKey).Result()
	if err != nil {
		t.Fatalf("state not found in redis: %v", err)
	}

	payload, err := decodeOAuthStatePayload(stateVal)
	if err != nil {
		t.Fatalf("failed to decode state payload: %v", err)
	}
	if payload.SourceName != testSourceName || payload.Purpose != OAuthPurposeLogin {
		t.Errorf("unexpected payload: %+v", payload)
	}

	// Case 2: Unknown Source Login URL
	w2 := performRequest(router, http.MethodGet, "/api/v1/oauth/login?source=nonexistent", nil, nil, nil)
	if w2.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown source, got %d", w2.Code)
	}
}

func TestAuthorize(t *testing.T) {
	initializeTestConfig()
	dbConn := setupTestDB(t)
	mockRedis := newMockRedisClient()

	// Setup GitHub active source
	dbConn.Create(&model.AuthSource{
		ID:                 101,
		Name:               "github",
		Type:               model.AuthSourceTypeOIDC,
		DisplayName:        "GitHub OAuth",
		IsActive:           true,
		ClientID:           "gh_client",
		ClientSecret:       "gh_secret",
		OpenIDDiscoveryURL: "https://github.com",
	})

	// Mock OIDC Discovery request
	httpMock := &http.Client{
		Transport: &mockRoundTripper{
			roundTripFunc: func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet && strings.Contains(req.URL.String(), "/.well-known/openid-configuration") {
					body := `{
						"issuer": "https://github.com",
						"authorization_endpoint": "https://github.com/login/oauth/authorize",
						"token_endpoint": "https://github.com/login/oauth/access_token",
						"jwks_uri": "https://github.com/oauth/keys",
						"response_types_supported": ["code"],
						"subject_types_supported": ["public"],
						"id_token_signing_alg_values_supported": ["RS256"]
					}`
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(body)),
						Header:     make(http.Header),
					}, nil
				}
				return nil, fmt.Errorf("unexpected request: %s", req.URL)
			},
		},
	}
	util.SetHTTPClient(httpMock)
	router := setupTestRouter(dbConn, mockRedis, httpMock)

	// Case 1: Active Source Authorize with purpose=bind
	w := performRequest(router, http.MethodGet, "/api/v1/oauth/github/authorize?purpose=bind", nil, nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data OAuthAuthorizeResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	parsedURL, _ := url.Parse(resp.Data.AuthorizeURL)
	state := parsedURL.Query().Get("state")
	redisKey := db.PrefixedKey(fmt.Sprintf(OAuthStateCacheKeyFormat, state))
	stateVal, _ := mockRedis.Get(context.Background(), redisKey).Result()
	payload, _ := decodeOAuthStatePayload(stateVal)

	if payload.SourceName != "github" || payload.Purpose != OAuthPurposeBind {
		t.Errorf("expected source github with purpose bind, got %+v", payload)
	}

	// Case 2: Inactive Source Authorize
	dbConn.Model(&model.AuthSource{}).Where("id = ?", 101).Update("is_active", false)
	w2 := performRequest(router, http.MethodGet, "/api/v1/oauth/github/authorize", nil, nil, nil)
	if w2.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for inactive source, got %d", w2.Code)
	}
}

func TestLogout(t *testing.T) {
	initializeTestConfig()
	dbConn := setupTestDB(t)
	mockRedis := newMockRedisClient()
	httpMock := &http.Client{}
	router := setupTestRouter(dbConn, mockRedis, httpMock)

	w := performRequest(router, http.MethodGet, "/api/v1/oauth/logout", nil, nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCallbackLoginAndUserInfo(t *testing.T) {
	initializeTestConfig()
	dbConn := setupTestDB(t)
	mockRedis := newMockRedisClient()
	seedTestAuthSource(t, dbConn)
	state := uuid.NewString()

	// 1. Mock the outgoing HTTP client for token exchange and user info fetching
	httpMock := newMockOIDCClient(testIssuerURL, testClientID, state, "88888", "test_oauth_user", "oauth@linux.do", "Oauth Test User")
	util.SetHTTPClient(httpMock)
	router := setupTestRouter(dbConn, mockRedis, httpMock)

	// 2. Setup state in Redis
	payloadValue, _ := encodeOAuthStatePayload(oauthStatePayload{
		SourceName: testSourceName,
		Purpose:    OAuthPurposeLogin,
	})
	mockRedis.Set(context.Background(), db.PrefixedKey(fmt.Sprintf(OAuthStateCacheKeyFormat, state)), payloadValue, OAuthStateCacheKeyExpiration)

	// 3. Trigger Callback (Login flow - new user)
	reqBody := fmt.Sprintf(`{"state":"%s","code":"test_auth_code"}`, state)
	w := performRequest(router, http.MethodPost, "/api/v1/oauth/callback", []byte(reqBody), map[string]string{
		"Content-Type": "application/json",
	}, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("callback failed with status %d, body: %s", w.Code, w.Body.String())
	}

	var callbackResp struct {
		Data OAuthCallbackResult `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &callbackResp)

	if callbackResp.Data.Status != "logged_in" {
		t.Errorf("expected logged_in status, got %s", callbackResp.Data.Status)
	}

	if callbackResp.Data.User.Username != "test_oauth_user" || callbackResp.Data.User.ID != 88888 {
		t.Errorf("unexpected user returned: %+v", callbackResp.Data.User)
	}

	// Verify user is created in database
	var user model.User
	if err := dbConn.First(&user, "id = ?", 88888).Error; err != nil {
		t.Fatalf("user was not created in DB: %v", err)
	}

	// Extract session cookie
	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == config.Config.App.SessionCookieName {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("session cookie not found in response")
	}

	// 4. Test GET user-info
	w2 := performRequest(router, http.MethodGet, "/api/v1/oauth/user-info", nil, nil, []*http.Cookie{sessionCookie})
	if w2.Code != http.StatusOK {
		t.Fatalf("failed to fetch user info, status %d", w2.Code)
	}

	// 5. Test Callback (Login flow - existing user, username collision check)
	state2 := uuid.NewString()
	mockRedis.Set(context.Background(), db.PrefixedKey(fmt.Sprintf(OAuthStateCacheKeyFormat, state2)), payloadValue, OAuthStateCacheKeyExpiration)

	// Callback with same username but different external ID (99999)
	httpMock2 := newMockOIDCClient(testIssuerURL, testClientID, state2, "99999", "test_oauth_user", "another@linux.do", "Another User")
	util.SetHTTPClient(httpMock2)

	// Create another router for this mock client
	router2 := setupTestRouter(dbConn, mockRedis, httpMock2)

	reqBody2 := fmt.Sprintf(`{"state":"%s","code":"test_auth_code"}`, state2)
	w3 := performRequest(router2, http.MethodPost, "/api/v1/oauth/callback", []byte(reqBody2), map[string]string{
		"Content-Type": "application/json",
	}, nil)

	if w3.Code != http.StatusOK {
		t.Fatalf("callback for collision failed: %d, body: %s", w3.Code, w3.Body.String())
	}

	var collisionResp struct {
		Data OAuthCallbackResult `json:"data"`
	}
	_ = json.Unmarshal(w3.Body.Bytes(), &collisionResp)

	if collisionResp.Data.User.Username != "test_oauth_user-1" {
		t.Errorf("expected collision renamed username, got %s", collisionResp.Data.User.Username)
	}

	t.Run("OIDC login when registration disabled - need bind", func(t *testing.T) {
		// Disable registration in database
		dbConn.Create(&model.SystemConfig{
			Key:   model.ConfigKeyRegistrationEnabled,
			Value: "false",
		})
		defer func() {
			dbConn.Where("key = ?", model.ConfigKeyRegistrationEnabled).Delete(&model.SystemConfig{})
		}()

		state4 := uuid.NewString()
		payloadValue4, _ := encodeOAuthStatePayload(oauthStatePayload{
			SourceName: testSourceName,
			Purpose:    OAuthPurposeLogin,
		})
		mockRedis.Set(context.Background(), db.PrefixedKey(fmt.Sprintf(OAuthStateCacheKeyFormat, state4)), payloadValue4, OAuthStateCacheKeyExpiration)

		httpMock4 := newMockOIDCClient(testIssuerURL, testClientID, state4, "77777", "need_bind_user", "needbind@linux.do", "Need Bind User")
		util.SetHTTPClient(httpMock4)
		router4 := setupTestRouter(dbConn, mockRedis, httpMock4)

		reqBody4 := fmt.Sprintf(`{"state":"%s","code":"test_auth_code"}`, state4)
		w4 := performRequest(router4, http.MethodPost, "/api/v1/oauth/callback", []byte(reqBody4), map[string]string{
			"Content-Type": "application/json",
		}, nil)

		if w4.Code != http.StatusOK {
			t.Fatalf("callback failed: %d, body: %s", w4.Code, w4.Body.String())
		}

		var needBindResp struct {
			Data OAuthCallbackResult `json:"data"`
		}
		_ = json.Unmarshal(w4.Body.Bytes(), &needBindResp)

		if needBindResp.Data.Status != "need_bind" {
			t.Errorf("expected status 'need_bind', got %s", needBindResp.Data.Status)
		}
		if needBindResp.Data.User != nil {
			t.Errorf("expected User to be nil, got %+v", needBindResp.Data.User)
		}
	})
}

func TestCallbackBind(t *testing.T) {
	initializeTestConfig()
	dbConn := setupTestDB(t)
	mockRedis := newMockRedisClient()

	// Create user
	user := model.User{
		ID:          777,
		Username:    "existing_member",
		Nickname:    "Existing Member",
		IsActive:    true,
		LastLoginAt: time.Now(),
	}
	dbConn.Create(&user)

	// Create auth source "github"
	dbConn.Create(&model.AuthSource{
		ID:                 2,
		Name:               "github",
		Type:               model.AuthSourceTypeOIDC,
		DisplayName:        "GitHub",
		IsActive:           true,
		ClientID:           "gh_client",
		ClientSecret:       "gh_secret",
		OpenIDDiscoveryURL: "https://github.com",
	})

	// Inject state in Redis with purpose=bind
	state := uuid.NewString()
	payloadValue, _ := encodeOAuthStatePayload(oauthStatePayload{
		SourceName: "github",
		Purpose:    OAuthPurposeBind,
	})
	mockRedis.Set(context.Background(), db.PrefixedKey(fmt.Sprintf(OAuthStateCacheKeyFormat, state)), payloadValue, OAuthStateCacheKeyExpiration)

	// Mock OIDC discovery, JWKS, and Token exchange for custom source (GitHub)
	httpMock := newMockOIDCClient("https://github.com", "gh_client", state, "github_user_123", "github_tester", "tester@github.com", "GitHub Tester")
	util.SetHTTPClient(httpMock)
	router := setupTestRouter(dbConn, mockRedis, httpMock)

	// Case 1: Bind attempt without session -> 401
	reqBody := fmt.Sprintf(`{"state":"%s","code":"test_code"}`, state)
	w1 := performRequest(router, http.MethodPost, "/api/v1/oauth/callback", []byte(reqBody), map[string]string{
		"Content-Type": "application/json",
	}, nil)

	if w1.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated bind, got %d, body: %s", w1.Code, w1.Body.String())
	}

	// Case 2: Bind success (authenticated)
	router.GET("/test-helper/login-777", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set(UserIDKey, uint64(777))
		_ = session.Save()
		c.String(200, "ok")
	})

	wLogin := performRequest(router, http.MethodGet, "/test-helper/login-777", nil, nil, nil)
	var activeCookie *http.Cookie
	for _, cookie := range wLogin.Result().Cookies() {
		if cookie.Name == config.Config.App.SessionCookieName {
			activeCookie = cookie
			break
		}
	}

	// Restore state key in redis
	mockRedis.Set(context.Background(), db.PrefixedKey(fmt.Sprintf(OAuthStateCacheKeyFormat, state)), payloadValue, OAuthStateCacheKeyExpiration)

	w2 := performRequest(router, http.MethodPost, "/api/v1/oauth/callback", []byte(reqBody), map[string]string{
		"Content-Type": "application/json",
	}, []*http.Cookie{activeCookie})

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for bind callback, got %d, body: %s", w2.Code, w2.Body.String())
	}

	var bindResult struct {
		Data OAuthCallbackResult `json:"data"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &bindResult)
	if bindResult.Data.Status != "bound" {
		t.Errorf("expected status bound, got %s", bindResult.Data.Status)
	}

	// Verify DB binding
	var binding model.ExternalAccount
	if err := dbConn.First(&binding, "user_id = ? AND external_id = ?", 777, "github_user_123").Error; err != nil {
		t.Fatalf("DB binding record not found: %v", err)
	}

	// Case 3: Bind already bound account to another user
	// Create another user
	user2 := model.User{
		ID:          888,
		Username:    "another_member",
		Nickname:    "Another Member",
		IsActive:    true,
		LastLoginAt: time.Now(),
	}
	dbConn.Create(&user2)

	router.GET("/test-helper/login-888", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set(UserIDKey, uint64(888))
		_ = session.Save()
		c.String(200, "ok")
	})

	wLogin2 := performRequest(router, http.MethodGet, "/test-helper/login-888", nil, nil, nil)
	var activeCookie2 *http.Cookie
	for _, cookie := range wLogin2.Result().Cookies() {
		if cookie.Name == config.Config.App.SessionCookieName {
			activeCookie2 = cookie
			break
		}
	}

	state3 := uuid.NewString()
	mockRedis.Set(context.Background(), db.PrefixedKey(fmt.Sprintf(OAuthStateCacheKeyFormat, state3)), payloadValue, OAuthStateCacheKeyExpiration)

	// Re-sign token for new state (since state serves as OIDC Nonce)
	httpMock3 := newMockOIDCClient("https://github.com", "gh_client", state3, "github_user_123", "github_tester", "tester@github.com", "GitHub Tester")
	util.SetHTTPClient(httpMock3)
	router3 := setupTestRouter(dbConn, mockRedis, httpMock3)

	reqBody3 := fmt.Sprintf(`{"state":"%s","code":"test_code"}`, state3)
	w3 := performRequest(router3, http.MethodPost, "/api/v1/oauth/callback", []byte(reqBody3), map[string]string{
		"Content-Type": "application/json",
	}, []*http.Cookie{activeCookie2})

	if w3.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for already bound account, got %d, body: %s", w3.Code, w3.Body.String())
	}
}

func TestExternalAccountsListAndDelete(t *testing.T) {
	initializeTestConfig()
	dbConn := setupTestDB(t)
	mockRedis := newMockRedisClient()
	httpMock := &http.Client{}
	router := setupTestRouter(dbConn, mockRedis, httpMock)

	// Create user and external accounts
	dbConn.Create(&model.User{
		ID:       555,
		Username: "account_holder",
		IsActive: true,
	})

	dbConn.Create(&model.AuthSource{
		ID:       10,
		Name:     "gitlab",
		Type:     model.AuthSourceTypeOIDC,
		IsActive: true,
	})

	dbConn.Create(&model.ExternalAccount{
		ID:               2001,
		AuthSourceID:     10,
		UserID:           555,
		ExternalID:       "gitlab_123",
		ExternalUsername: "gitlab_user",
	})

	router.GET("/test-helper/login-555", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set(UserIDKey, uint64(555))
		_ = session.Save()
		c.String(200, "ok")
	})

	wLogin := performRequest(router, http.MethodGet, "/test-helper/login-555", nil, nil, nil)
	var activeCookie *http.Cookie
	for _, cookie := range wLogin.Result().Cookies() {
		if cookie.Name == config.Config.App.SessionCookieName {
			activeCookie = cookie
			break
		}
	}

	// 1. List accounts
	wList := performRequest(router, http.MethodGet, "/api/v1/oauth/external-accounts", nil, nil, []*http.Cookie{activeCookie})
	if wList.Code != http.StatusOK {
		t.Fatalf("failed to list external accounts: %d", wList.Code)
	}

	var listResp struct {
		Data []model.ExternalAccountView `json:"data"`
	}
	_ = json.Unmarshal(wList.Body.Bytes(), &listResp)
	if len(listResp.Data) != 1 || listResp.Data[0].ExternalUsername != "gitlab_user" {
		t.Errorf("unexpected list response: %+v", listResp.Data)
	}

	// 2. Delete/Unbind account
	wDelete := performRequest(router, http.MethodPost, "/api/v1/oauth/external-accounts/2001/delete", nil, nil, []*http.Cookie{activeCookie})
	if wDelete.Code != http.StatusOK {
		t.Fatalf("failed to delete external account binding: %d, body: %s", wDelete.Code, wDelete.Body.String())
	}

	var count int64
	dbConn.Model(&model.ExternalAccount{}).Where("id = ?", 2001).Count(&count)
	if count != 0 {
		t.Error("binding record was not deleted from DB")
	}
}
