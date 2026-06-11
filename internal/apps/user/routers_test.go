// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/Rain-kl/Wavelet/internal/config"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func setupUserTestRouter(t *testing.T) *gin.Engine {
	t.Helper()

	oldCookieName := config.Config.App.SessionCookieName
	oldSecret := config.Config.App.SessionSecret
	oldDomain := config.Config.App.SessionDomain
	oldSecure := config.Config.App.SessionSecure
	oldHTTPOnly := config.Config.App.SessionHTTPOnly
	t.Cleanup(func() {
		config.Config.App.SessionCookieName = oldCookieName
		config.Config.App.SessionSecret = oldSecret
		config.Config.App.SessionDomain = oldDomain
		config.Config.App.SessionSecure = oldSecure
		config.Config.App.SessionHTTPOnly = oldHTTPOnly
	})

	config.Config.App.SessionCookieName = "test_session_id"
	config.Config.App.SessionSecret = "test_session_secret"
	config.Config.App.SessionDomain = ""
	config.Config.App.SessionSecure = false
	config.Config.App.SessionHTTPOnly = true

	gin.SetMode(gin.TestMode)
	r := gin.New()
	store := cookie.NewStore([]byte(config.Config.App.SessionSecret))
	store.Options(util.GetSessionOptions(3600))
	r.Use(sessions.Sessions(config.Config.App.SessionCookieName, store))

	api := r.Group("/api/v1")
	api.POST("/user/register", Register)
	api.POST("/user/login", Login)
	api.GET("/user-info", oauth.LoginRequired(), oauth.UserInfo)
	return r
}

func performUserRequest(r http.Handler, method, path string, body []byte, cookies []*http.Cookie) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}

	req, _ := http.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func sessionCookieFromResponse(t *testing.T, w *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()

	for _, c := range w.Result().Cookies() {
		if c.Name == config.Config.App.SessionCookieName {
			return c
		}
	}
	t.Fatalf("sessionCookieFromResponse() did not find %q cookie", config.Config.App.SessionCookieName)
	return nil
}

func basicUserInfoFromResponse(t *testing.T, w *httptest.ResponseRecorder) oauth.BasicUserInfo {
	t.Helper()

	var resp util.ResponseAny
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("basicUserInfoFromResponse() decode response failed: %v", err)
	}
	if resp.ErrorMsg != "" {
		t.Fatalf("basicUserInfoFromResponse() error_msg = %q, want empty", resp.ErrorMsg)
	}
	data, _ := json.Marshal(resp.Data)
	var info oauth.BasicUserInfo
	if err := json.Unmarshal(data, &info); err != nil {
		t.Fatalf("basicUserInfoFromResponse() decode data failed: %v", err)
	}
	return info
}

func TestEmailCooldownKeyIncludesScene(t *testing.T) {
	email := "user@example.com"

	loginKey := getEmailCooldownKey("login", email)
	registerKey := getEmailCooldownKey("register", email)
	if loginKey == registerKey {
		t.Errorf("getEmailCooldownKey(%q, %q) = %q, want different key from register scene", "login", email, loginKey)
	}
	if want := "email_code:cooldown:login:user@example.com"; loginKey != want {
		t.Errorf("getEmailCooldownKey(%q, %q) = %q, want %q", "login", email, loginKey, want)
	}
}

func TestGenerateVerificationCode(t *testing.T) {
	code, err := generateVerificationCode()
	if err != nil {
		t.Fatalf("generateVerificationCode() error = %v, want nil", err)
	}
	if len(code) != 6 {
		t.Fatalf("generateVerificationCode() length = %d, want 6. Code: %q", len(code), code)
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			t.Fatalf("generateVerificationCode() = %q, want only digits", code)
		}
	}
}

func TestRegisterCreatesAuthenticatedEncryptedUser(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	router := setupUserTestRouter(t)
	payload := registerRequest{
		Username: "newuser",
		Password: "newpassword123",
		Nickname: "New User",
	}
	body, _ := json.Marshal(payload)

	w := performUserRequest(router, http.MethodPost, "/api/v1/user/register", body, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("Register(%q) status = %d, want %d. Body: %s", payload.Username, w.Code, http.StatusOK, w.Body.String())
	}
	info := basicUserInfoFromResponse(t, w)
	if info.NeedChangePassword {
		t.Errorf("Register(%q) need_change_password = true, want false", payload.Username)
	}

	var dbUser model.User
	if err := dbConn.Where("username = ?", payload.Username).First(&dbUser).Error; err != nil {
		t.Fatalf("Register(%q) db lookup failed: %v", payload.Username, err)
	}
	if dbUser.ID < 1000 {
		t.Errorf("Register(%q) user ID = %d, want generated snowflake ID", payload.Username, dbUser.ID)
	}
	if !dbUser.IsPasswordEncrypted() {
		t.Errorf("Register(%q) stored plaintext password, want encrypted password", payload.Username)
	}
	if !dbUser.CheckPassword(payload.Password) {
		t.Errorf("Register(%q) stored password does not match original password", payload.Username)
	}

	sessionCookie := sessionCookieFromResponse(t, w)
	w = performUserRequest(router, http.MethodGet, "/api/v1/user-info", nil, []*http.Cookie{sessionCookie})
	if w.Code != http.StatusOK {
		t.Fatalf("UserInfo() after Register(%q) status = %d, want %d. Body: %s", payload.Username, w.Code, http.StatusOK, w.Body.String())
	}
}

func TestLoginRequiresPasswordChangeForInitialPlaintextAdmin(t *testing.T) {
	dbConn, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	const (
		adminID       = uint64(1)
		adminUsername = "admin"
		adminPassword = "12345678"
	)
	now := time.Now()
	if err := dbConn.Exec(
		`INSERT INTO w_users (id, username, password, nickname, is_active, is_admin, last_login_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		adminID,
		adminUsername,
		adminPassword,
		"Administrator",
		true,
		true,
		now,
		now,
		now,
	).Error; err != nil {
		t.Fatalf("seed initial admin failed: %v", err)
	}

	router := setupUserTestRouter(t)
	payload := loginRequest{
		Username: adminUsername,
		Password: adminPassword,
	}
	body, _ := json.Marshal(payload)

	w := performUserRequest(router, http.MethodPost, "/api/v1/user/login", body, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("Login(%q) status = %d, want %d. Body: %s", adminUsername, w.Code, http.StatusOK, w.Body.String())
	}
	info := basicUserInfoFromResponse(t, w)
	if !info.NeedChangePassword {
		t.Errorf("Login(%q) need_change_password = false, want true", adminUsername)
	}

	var dbUser model.User
	if err := dbConn.Where("username = ?", adminUsername).First(&dbUser).Error; err != nil {
		t.Fatalf("Login(%q) db lookup failed: %v", adminUsername, err)
	}
	if dbUser.ID != adminID {
		t.Errorf("Login(%q) user ID = %d, want %d", adminUsername, dbUser.ID, adminID)
	}
	if dbUser.IsPasswordEncrypted() {
		t.Errorf("Login(%q) encrypted password during login, want plaintext until password change", adminUsername)
	}
	if !dbUser.CheckPassword(adminPassword) {
		t.Errorf("Login(%q) stored password does not match original password", adminUsername)
	}

	sessionCookie := sessionCookieFromResponse(t, w)
	w = performUserRequest(router, http.MethodGet, "/api/v1/user-info", nil, []*http.Cookie{sessionCookie})
	if w.Code != http.StatusOK {
		t.Fatalf("UserInfo() after Login(%q) status = %d, want %d. Body: %s", adminUsername, w.Code, http.StatusOK, w.Body.String())
	}
	info = basicUserInfoFromResponse(t, w)
	if !info.NeedChangePassword {
		t.Errorf("UserInfo() after Login(%q) need_change_password = false, want true", adminUsername)
	}
}
