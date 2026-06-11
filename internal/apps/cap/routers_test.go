// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package cap

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/testhelper"
	"github.com/Rain-kl/Wavelet/internal/util"
	capUtil "github.com/Rain-kl/Wavelet/internal/util/cap"
	"github.com/gin-gonic/gin"
)

func TestCapEndpointsAndMiddleware(t *testing.T) {
	sqliteDB, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Mount CAPTCHA API endpoints
	capGroup := r.Group("/api/cap")
	{
		capGroup.POST("/challenge", Challenge)
		capGroup.POST("/redeem", Redeem)
	}

	// Login endpoint with CAPTCHA middleware
	r.POST("/api/v1/user/login", VerifyMiddleware(capUtil.GetDefaultManager(), "login", func() bool {
		enabled, err := model.GetBoolByKey(context.Background(), model.ConfigKeyCapLoginEnabled)
		if err != nil {
			return false
		}
		return enabled
	}), func(c *gin.Context) {
		c.JSON(http.StatusOK, util.OK("login success"))
	})

	// 1. Test challenge generation
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/cap/challenge", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var challengeResp capUtil.ChallengeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &challengeResp); err != nil {
		t.Fatalf("failed to unmarshal challenge response: %v", err)
	}

	if challengeResp.Token == "" {
		t.Fatalf("expected token in challenge response")
	}

	// 2. Test login with CAPTCHA disabled (should pass)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/user/login", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK when CAPTCHA is disabled, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 3. Enable CAPTCHA in DB
	err := sqliteDB.Model(&model.SystemConfig{}).Where("key = ?", model.ConfigKeyCapLoginEnabled).Update("value", "true").Error
	if err != nil {
		t.Fatalf("failed to enable cap_login_enabled in DB: %v", err)
	}
	// Update cache
	var sysCfg model.SystemConfig
	sqliteDB.Where("key = ?", model.ConfigKeyCapLoginEnabled).First(&sysCfg)
	_ = db.HSetJSON(context.Background(), model.SystemConfigRedisHashKey, model.ConfigKeyCapLoginEnabled, &sysCfg)

	// 4. Test login with CAPTCHA enabled but no header (should be blocked)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/user/login", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 5. Solve the challenge
	solutions := capUtil.Solve(challengeResp.Token, challengeResp.Challenge.C, challengeResp.Challenge.S, challengeResp.Challenge.D)

	// 6. Redeem solutions
	redeemReqPayload := redeemRequest{
		Token:     challengeResp.Token,
		Solutions: solutions,
	}
	bodyBytes, _ := json.Marshal(redeemReqPayload)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/cap/redeem", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for redeem, got %d. Body: %s", w.Code, w.Body.String())
	}

	var redeemResp capUtil.RedeemResponse
	if err := json.Unmarshal(w.Body.Bytes(), &redeemResp); err != nil {
		t.Fatalf("failed to unmarshal redeem response: %v", err)
	}

	if !redeemResp.Success || redeemResp.Token == "" {
		t.Fatalf("redeem failed or returned empty token: %+v", redeemResp)
	}

	// 7. Login with valid redeem token (should pass)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/user/login", nil)
	req.Header.Set("X-Cap-Token", redeemResp.Token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with valid cap token, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 8. Replay attack: Login with the same redeem token again (should be blocked as it is single-use)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/user/login", nil)
	req.Header.Set("X-Cap-Token", redeemResp.Token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized on replayed token, got %d. Body: %s", w.Code, w.Body.String())
	}
}
