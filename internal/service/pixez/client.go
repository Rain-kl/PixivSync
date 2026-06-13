// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

package pixez

import (
	"bytes"
	"context"
	//nolint:gosec // MD5 is required by the official Pixiv API signature format
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/logger"
	"github.com/Rain-kl/Wavelet/internal/model"
	"gorm.io/gorm"
)

const (
	pixivAppUserAgent         = "PixivAndroidApp/5.0.166 (Android 16; PKX110)"
	pixivAppVersion           = "5.0.166"
	pixivAppOS                = "Android"
	pixivAppOSVersion         = "Android 16"
	pixivAPIHost              = "app-api.pixiv.net"
	defaultPixivClientTimeout = 45 * time.Second
	detectContentBytes        = 512
	novelJSONSubmatchCount    = 2
	maxPixivFileSize          = 512 << 20
)

var novelJSONRegex = regexp.MustCompile(`(?s)novel:\s*(\{.*?\}),\s*\n\s*isOwnWork`)

// Client centralizes official Pixiv API requests used by PixEz migration code.
type Client struct {
	HTTPClient *http.Client
}

// DefaultClient is the shared PixEz Pixiv client.
var DefaultClient = NewClient(nil)

// NewClient creates a Pixiv client with a default timeout when no HTTP client is supplied.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultPixivClientTimeout}
	}
	return &Client{HTTPClient: httpClient}
}

// GetIllustDetail fetches Pixiv illustration detail and validates the payload.
func (c *Client) GetIllustDetail(ctx context.Context, user model.PixezPixivUser, illustID int64) ([]byte, IllustDetail, error) {
	reqURL := fmt.Sprintf("https://%s/v1/illust/detail?filter=for_android&illust_id=%d", pixivAPIHost, illustID)
	var detail IllustDetail
	data, err := c.getJSONWithAuth(ctx, user, reqURL, &detail)
	if err != nil {
		return nil, detail, err
	}
	if detail.Illust.ID == 0 {
		return nil, detail, fmt.Errorf("pixiv illust detail response missing illust.id for illust_id=%d", illustID)
	}
	return data, detail, nil
}

// GetNovelDetail fetches Pixiv novel detail and validates the payload.
func (c *Client) GetNovelDetail(ctx context.Context, user model.PixezPixivUser, novelID int64) ([]byte, NovelDetail, error) {
	reqURL := fmt.Sprintf("https://%s/v2/novel/detail?novel_id=%d", pixivAPIHost, novelID)
	var detail NovelDetail
	data, err := c.getJSONWithAuth(ctx, user, reqURL, &detail)
	if err != nil {
		return nil, detail, err
	}
	if detail.Novel.ID == 0 {
		return nil, detail, fmt.Errorf("pixiv novel detail response missing novel.id for novel_id=%d", novelID)
	}
	return data, detail, nil
}

// GetNovelText fetches Pixiv webview novel content and extracts the embedded JSON.
func (c *Client) GetNovelText(ctx context.Context, user model.PixezPixivUser, novelID int64) ([]byte, NovelWebContent, error) {
	reqURL := fmt.Sprintf("https://%s/webview/v2/novel?id=%d", pixivAPIHost, novelID)
	data, status, err := c.doPixivGet(ctx, reqURL, user.AccessToken)
	if err != nil {
		return nil, NovelWebContent{}, err
	}
	if status != http.StatusOK {
		return nil, NovelWebContent{}, fmt.Errorf("request=%s status=%d response=%s", reqURL, status, string(data))
	}

	matches := novelJSONRegex.FindSubmatch(data)
	if len(matches) < novelJSONSubmatchCount {
		return nil, NovelWebContent{}, fmt.Errorf("novel JSON not found in webview response for novel_id=%d", novelID)
	}

	novelJSONBytes := matches[1]
	var content NovelWebContent
	if err := json.Unmarshal(novelJSONBytes, &content); err != nil {
		return nil, NovelWebContent{}, fmt.Errorf("parse novel webview JSON failed: %w", err)
	}
	return novelJSONBytes, content, nil
}

// GetBookmarkIllusts fetches one page of Pixiv bookmark illustrations.
func (c *Client) GetBookmarkIllusts(ctx context.Context, user model.PixezPixivUser, reqURL string) ([]byte, BookmarkIllustResponse, error) {
	var payload BookmarkIllustResponse
	data, err := c.getJSONWithAuth(ctx, user, reqURL, &payload)
	return data, payload, err
}

// GetBookmarkNovels fetches one page of Pixiv bookmark novels.
func (c *Client) GetBookmarkNovels(ctx context.Context, user model.PixezPixivUser, reqURL string) ([]byte, BookmarkNovelResponse, error) {
	var payload BookmarkNovelResponse
	data, err := c.getJSONWithAuth(ctx, user, reqURL, &payload)
	return data, payload, err
}

// InitialBookmarkIllustURL returns the first Pixiv bookmark illustration URL.
func InitialBookmarkIllustURL(userID string, restrict string) string {
	values := url.Values{}
	values.Set("user_id", userID)
	values.Set("restrict", restrict)
	return "https://" + pixivAPIHost + "/v1/user/bookmarks/illust?" + values.Encode()
}

// InitialBookmarkNovelURL returns the first Pixiv bookmark novel URL.
func InitialBookmarkNovelURL(userID string, restrict string) string {
	values := url.Values{}
	values.Set("user_id", userID)
	values.Set("restrict", restrict)
	return "https://" + pixivAPIHost + "/v1/user/bookmarks/novel?" + values.Encode()
}

// DownloadFile fetches one Pixiv image into memory for upload storage registration.
func (c *Client) DownloadFile(ctx context.Context, fileURL string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Referer", "https://app-api.pixiv.net/")
	req.Header.Set("User-Agent", pixivAppUserAgent)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()

	data, readErr := io.ReadAll(io.LimitReader(resp.Body, maxPixivFileSize+1))
	if readErr != nil {
		return data, "", readErr
	}
	if int64(len(data)) > maxPixivFileSize {
		return nil, "", fmt.Errorf("pixiv file exceeds max size: %s", fileURL)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("request=%s status=%d response=%s", fileURL, resp.StatusCode, string(data))
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data[:min(len(data), detectContentBytes)])
	}
	return data, contentType, nil
}

func (c *Client) getJSONWithAuth(ctx context.Context, user model.PixezPixivUser, reqURL string, target any) ([]byte, error) {
	data, status, err := c.requestJSON(ctx, reqURL, user.AccessToken, target)
	if err == nil {
		return data, nil
	}
	if !isAuthError(status, data) {
		return nil, err
	}

	freshData, ok, err := c.tryLatestAccessToken(ctx, user, reqURL, target)
	if err != nil || ok {
		return freshData, err
	}

	latestUser := user
	if latest, found := latestPixivUser(ctx, user.PixivUserID); found {
		latestUser = latest
	}
	newAccess, refreshErr := c.RefreshPixivToken(ctx, latestUser.PixivUserID, latestUser.RefreshToken)
	if refreshErr != nil {
		return nil, fmt.Errorf("token refresh failed: %w", refreshErr)
	}

	data, _, err = c.requestJSON(ctx, reqURL, newAccess, target)
	return data, err
}

func (c *Client) requestJSON(ctx context.Context, reqURL string, accessToken string, target any) ([]byte, int, error) {
	data, status, err := c.doPixivGet(ctx, reqURL, accessToken)
	if err != nil {
		return data, status, err
	}
	if status != http.StatusOK {
		return data, status, fmt.Errorf("request=%s status=%d response=%s", reqURL, status, string(data))
	}
	if err := json.Unmarshal(data, target); err != nil {
		return nil, status, fmt.Errorf("parse Pixiv response request=%s failed: %w", reqURL, err)
	}
	return data, status, nil
}

func (c *Client) tryLatestAccessToken(ctx context.Context, user model.PixezPixivUser, reqURL string, target any) ([]byte, bool, error) {
	latest, ok := latestPixivUser(ctx, user.PixivUserID)
	if !ok || latest.AccessToken == "" || latest.AccessToken == user.AccessToken {
		return nil, false, nil
	}
	data, status, err := c.requestJSON(ctx, reqURL, latest.AccessToken, target)
	if err == nil {
		return data, true, nil
	}
	if isAuthError(status, data) {
		return nil, false, nil
	}
	return nil, true, err
}

// RefreshPixivToken refreshes Pixiv credentials and persists the latest tokens.
func (c *Client) RefreshPixivToken(ctx context.Context, userID string, refreshToken string) (string, error) {
	timeStr := pixivClientTime()
	form := url.Values{}
	form.Set("client_id", "MOBrBDS8blbauoSck0ZfDbtuzpyT")
	form.Set("client_secret", "lsACyCD94FhDUtGTXi3QzcFE2uU1hqtDaKeqrdwj")
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("include_policy", "true")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth.secure.pixiv.net/auth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	setPixivAppHeaders(req, "")
	req.Header.Set("X-Client-Time", timeStr)
	req.Header.Set("X-Client-Hash", pixivClientHash(timeStr))

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return "", readErr
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request=%s status=%d response=%s", req.URL.String(), resp.StatusCode, string(bodyBytes))
	}

	var tokenResp struct {
		Response struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"response"`
	}
	if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
		return "", err
	}
	if tokenResp.Response.AccessToken == "" {
		return "", fmt.Errorf("received empty access token")
	}

	updates := map[string]any{
		"access_token": tokenResp.Response.AccessToken,
		keyUpdatedAt:   time.Now(),
	}
	if tokenResp.Response.RefreshToken != "" {
		updates["refresh_token"] = tokenResp.Response.RefreshToken
	}
	if err := db.DB(ctx).Model(&model.PixezPixivUser{}).Where("pixiv_user_id = ?", userID).Updates(updates).Error; err != nil {
		logger.ErrorF(ctx, "[PixEz] failed to persist refreshed Pixiv token user_id=%s: %v", userID, err)
	}
	return tokenResp.Response.AccessToken, nil
}

// AddPixivUserByRefreshToken exchanges a Pixiv refresh token for user credentials and creates or updates the user in the database.
func (c *Client) AddPixivUserByRefreshToken(ctx context.Context, refreshToken string) (*model.PixezPixivUser, error) {
	timeStr := pixivClientTime()
	form := url.Values{}
	form.Set("client_id", "MOBrBDS8blbauoSck0ZfDbtuzpyT")
	form.Set("client_secret", "lsACyCD94FhDUtGTXi3QzcFE2uU1hqtDaKeqrdwj")
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("include_policy", "true")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth.secure.pixiv.net/auth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	setPixivAppHeaders(req, "")
	req.Header.Set("X-Client-Time", timeStr)
	req.Header.Set("X-Client-Hash", pixivClientHash(timeStr))

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, readErr
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request=%s status=%d response=%s", req.URL.String(), resp.StatusCode, string(bodyBytes))
	}

	var tokenResp struct {
		Response struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			User         struct {
				ID               string `json:"id"`
				Name             string `json:"name"`
				Account          string `json:"account"`
				MailAddress      string `json:"mail_address"`
				ProfileImageUrls struct {
					Px170x170 string `json:"px_170x170"`
					Px50x50   string `json:"px_50x50"`
					Px16x16   string `json:"px_16x16"`
				} `json:"profile_image_urls"`
				IsPremium        bool `json:"is_premium"`
				XRestrict        int  `json:"x_restrict"`
				IsMailAuthorized bool `json:"is_mail_authorized"`
			} `json:"user"`
		} `json:"response"`
	}

	if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
		return nil, err
	}
	if tokenResp.Response.AccessToken == "" {
		return nil, fmt.Errorf("received empty access token")
	}

	resUser := tokenResp.Response.User
	if resUser.ID == "" {
		return nil, fmt.Errorf("received empty pixiv user id")
	}

	userImg := resUser.ProfileImageUrls.Px170x170
	if userImg == "" {
		userImg = resUser.ProfileImageUrls.Px50x50
	}
	if userImg == "" {
		userImg = resUser.ProfileImageUrls.Px16x16
	}

	isPremiumVal := 0
	if resUser.IsPremium {
		isPremiumVal = 1
	}
	isMailAuthVal := 0
	if resUser.IsMailAuthorized {
		isMailAuthVal = 1
	}

	newRefreshToken := tokenResp.Response.RefreshToken
	if newRefreshToken == "" {
		newRefreshToken = refreshToken
	}

	var existing model.PixezPixivUser
	tx := db.DB(ctx)
	findErr := tx.Where("pixiv_user_id = ?", resUser.ID).First(&existing).Error
	switch {
	case findErr == nil:
		// Update existing record
		existing.Name = resUser.Name
		existing.Account = resUser.Account
		existing.MailAddress = resUser.MailAddress
		existing.UserImage = userImg
		existing.AccessToken = tokenResp.Response.AccessToken
		existing.RefreshToken = newRefreshToken
		existing.IsPremium = isPremiumVal
		existing.XRestrict = resUser.XRestrict
		existing.IsMailAuthorized = isMailAuthVal
		existing.UpdatedAt = time.Now()
		if saveErr := tx.Save(&existing).Error; saveErr != nil {
			return nil, saveErr
		}
		return &existing, nil
	case errors.Is(findErr, gorm.ErrRecordNotFound):
		// Create new record
		user := &model.PixezPixivUser{
			PixivUserID:      resUser.ID,
			Name:             resUser.Name,
			Account:          resUser.Account,
			MailAddress:      resUser.MailAddress,
			UserImage:        userImg,
			AccessToken:      tokenResp.Response.AccessToken,
			RefreshToken:     newRefreshToken,
			IsPremium:        isPremiumVal,
			XRestrict:        resUser.XRestrict,
			IsMailAuthorized: isMailAuthVal,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}
		if createErr := tx.Create(user).Error; createErr != nil {
			return nil, createErr
		}
		return user, nil
	default:
		return nil, findErr
	}
}

// GetUserProfile fetches Pixiv user profile detail from Pixiv API.
func (c *Client) GetUserProfile(ctx context.Context, user model.PixezPixivUser, targetUserID string) ([]byte, error) {
	reqURL := fmt.Sprintf("https://%s/v1/user/detail?filter=for_android&user_id=%s", pixivAPIHost, targetUserID)
	var raw map[string]any
	data, err := c.getJSONWithAuth(ctx, user, reqURL, &raw)
	if err != nil {
		return nil, err
	}
	return data, nil
}


func (c *Client) doPixivGet(ctx context.Context, reqURL string, accessToken string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, 0, err
	}
	setPixivAppHeaders(req, accessToken)
	req.Header.Set("Host", pixivAPIHost)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return data, resp.StatusCode, readErr
	}
	return data, resp.StatusCode, nil
}

func (c *Client) httpClient() *http.Client {
	if c != nil && c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func latestPixivUser(ctx context.Context, userID string) (model.PixezPixivUser, bool) {
	var user model.PixezPixivUser
	if err := db.DB(ctx).Where("pixiv_user_id = ?", userID).First(&user).Error; err != nil {
		return model.PixezPixivUser{}, false
	}
	return user, true
}

func setPixivAppHeaders(req *http.Request, accessToken string) {
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	timeStr := pixivClientTime()
	req.Header.Set("X-Client-Time", timeStr)
	req.Header.Set("X-Client-Hash", pixivClientHash(timeStr))
	req.Header.Set("User-Agent", pixivAppUserAgent)
	req.Header.Set("App-OS", pixivAppOS)
	req.Header.Set("App-OS-Version", pixivAppOSVersion)
	req.Header.Set("App-Version", pixivAppVersion)
}

func pixivClientTime() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05") + "+00:00"
}

func pixivClientHash(t string) string {
	hashSalt := "28c1fdd170a5204386cb1313c7077b34f83e4aaf4aa829ce78c231e05b0bae2c"
	//nolint:gosec // MD5 is required by the official Pixiv API signature format
	h := md5.Sum([]byte(t + hashSalt))
	return hex.EncodeToString(h[:])
}

func isAuthError(status int, body []byte) bool {
	if status == http.StatusUnauthorized {
		return true
	}
	return status == http.StatusBadRequest && bytes.Contains(body, []byte("invalid_grant"))
}

// IsLimitUnknownIllust returns true for Pixiv hidden placeholder illustration payloads.
func IsLimitUnknownIllust(illust Illust) bool {
	return strings.Contains(illust.ImageUrls.SquareMedium, limitUnknownIllust) ||
		strings.Contains(illust.ImageUrls.Medium, limitUnknownIllust) ||
		strings.Contains(illust.ImageUrls.Large, limitUnknownIllust) ||
		strings.Contains(illust.MetaSinglePage.OriginalImageURL, limitUnknownIllust)
}

// IsLimitUnknownNovel returns true for Pixiv hidden placeholder novel payloads.
func IsLimitUnknownNovel(novel BookmarkNovel) bool {
	return strings.Contains(novel.ImageUrls.SquareMedium, limitUnknownNovel) ||
		strings.Contains(novel.ImageUrls.Medium, limitUnknownNovel) ||
		strings.Contains(novel.ImageUrls.Large, limitUnknownNovel)
}

// CollectIllustImageURLs returns unique original image URLs from a detail payload.
func CollectIllustImageURLs(detail IllustDetail) []string {
	seen := make(map[string]bool)
	urls := make([]string, 0, max(1, len(detail.Illust.MetaPages)))
	add := func(v string) {
		if v == "" || seen[v] {
			return
		}
		seen[v] = true
		urls = append(urls, v)
	}

	add(detail.Illust.MetaSinglePage.OriginalImageURL)
	for _, page := range detail.Illust.MetaPages {
		add(page.ImageUrls.Original)
	}
	return urls
}
