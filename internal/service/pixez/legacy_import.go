/*
Copyright 2026 linux.do
Modified by Arctel.net, 2026

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pixez

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type legacyMirrorTask struct {
	ID              string
	TargetType      string
	TargetID        int64
	Status          string
	RequestURLsJSON string
	RetryURLsJSON   string
	ErrorMessage    string
	TotalCount      int
	SuccessCount    int
	FailedCount     int
}

func (legacyMirrorTask) TableName() string { return "mirror_tasks" }

type legacyImageFileRecord struct {
	URL      string `json:"url"`
	Path     string `json:"path"`
	Filename string `json:"filename"`
}

// ImportLegacyServer imports data from the legacy server SQLite database.
func ImportLegacyServer(ctx context.Context, req ImportLegacyRequest) (ImportLegacySummary, error) {
	if req.SQLitePath == "" {
		req.SQLitePath = "server/pixez-sync.db"
	}
	if req.MirrorDir == "" {
		req.MirrorDir = "server/data/mirror"
	}
	req.SQLitePath = fallbackSiblingPath(req.SQLitePath)
	req.MirrorDir = fallbackSiblingPath(req.MirrorDir)

	summary := ImportLegacySummary{DryRun: req.DryRun, StartedAt: time.Now()}
	legacyDB, err := gorm.Open(sqlite.Open(req.SQLitePath), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return summary, fmt.Errorf("open legacy SQLite database: %w", err)
	}
	sqlDB, err := legacyDB.DB()
	if err == nil {
		defer func() { _ = sqlDB.Close() }()
	}

	if err := importLegacyUsers(ctx, legacyDB, req.DryRun, &summary); err != nil {
		return summary, err
	}
	if err := importLegacySyncData(ctx, legacyDB, req.DryRun, &summary); err != nil {
		return summary, err
	}
	if err := importLegacyBookmarks(ctx, legacyDB, req.DryRun, &summary); err != nil {
		return summary, err
	}
	if err := importLegacyMirrors(ctx, legacyDB, req.MirrorDir, req.DryRun, &summary); err != nil {
		return summary, err
	}

	summary.FinishedAt = time.Now()
	return summary, nil
}

func importLegacyUsers(ctx context.Context, legacyDB *gorm.DB, dryRun bool, summary *ImportLegacySummary) error {
	var users []model.PixezPixivUser
	if err := legacyDB.Find(&users).Error; err != nil {
		return fmt.Errorf("read legacy pixiv_users: %w", err)
	}
	summary.PixivUsers = len(users)
	if dryRun || len(users) == 0 {
		return nil
	}
	return db.DB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "pixiv_user_id"}},
		UpdateAll: true,
	}).Create(&users).Error
}

func importLegacySyncData(ctx context.Context, legacyDB *gorm.DB, dryRun bool, summary *ImportLegacySummary) error {
	userIDs, err := legacyPixivUserIDs(legacyDB)
	if err != nil {
		return err
	}
	if dryRun {
		total, err := countLegacySyncRows(legacyDB)
		summary.SyncRows = total
		return err
	}

	return db.DB(ctx).Transaction(func(tx *gorm.DB) error {
		for _, userID := range userIDs {
			if err := deleteLegacySyncUserRows(tx, userID); err != nil {
				return err
			}
		}
		if err := copyLegacyRowsIntoSummary[model.PixezBanComment](legacyDB, tx, summary); err != nil {
			return err
		}
		if err := copyLegacyRowsIntoSummary[model.PixezBanIllust](legacyDB, tx, summary); err != nil {
			return err
		}
		if err := copyLegacyRowsIntoSummary[model.PixezBanTag](legacyDB, tx, summary); err != nil {
			return err
		}
		if err := copyLegacyRowsIntoSummary[model.PixezBanUser](legacyDB, tx, summary); err != nil {
			return err
		}
		if err := copyLegacyRowsIntoSummary[model.PixezIllustHistory](legacyDB, tx, summary); err != nil {
			return err
		}
		if err := copyLegacyRowsIntoSummary[model.PixezNovelHistory](legacyDB, tx, summary); err != nil {
			return err
		}
		if err := copyLegacyRowsIntoSummary[model.PixezTagHistory](legacyDB, tx, summary); err != nil {
			return err
		}
		return nil
	})
}

func legacyPixivUserIDs(legacyDB *gorm.DB) ([]string, error) {
	var rows []struct {
		PixivUserID string `gorm:"column:pixiv_user_id"`
	}
	if err := legacyDB.Raw(`
SELECT pixiv_user_id FROM pixiv_users
UNION SELECT pixiv_user_id FROM ban_comments
UNION SELECT pixiv_user_id FROM ban_illusts
UNION SELECT pixiv_user_id FROM ban_tags
UNION SELECT pixiv_user_id FROM ban_users
UNION SELECT pixiv_user_id FROM illust_histories
UNION SELECT pixiv_user_id FROM novel_histories
UNION SELECT pixiv_user_id FROM tag_histories
`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	userIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.PixivUserID != "" {
			userIDs = append(userIDs, row.PixivUserID)
		}
	}
	return userIDs, nil
}

func deleteLegacySyncUserRows(tx *gorm.DB, userID string) error {
	models := []any{
		&model.PixezBanComment{},
		&model.PixezBanIllust{},
		&model.PixezBanTag{},
		&model.PixezBanUser{},
		&model.PixezIllustHistory{},
		&model.PixezNovelHistory{},
		&model.PixezTagHistory{},
	}
	for _, m := range models {
		if err := tx.Where("pixiv_user_id = ?", userID).Delete(m).Error; err != nil {
			return err
		}
	}
	return nil
}

func copyLegacyRows[T any](legacyDB *gorm.DB, tx *gorm.DB) (int, error) {
	var rows []T
	if err := legacyDB.Find(&rows).Error; err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	if err := tx.Create(&rows).Error; err != nil {
		return 0, err
	}
	return len(rows), nil
}

func copyLegacyRowsIntoSummary[T any](legacyDB *gorm.DB, tx *gorm.DB, summary *ImportLegacySummary) error {
	n, err := copyLegacyRows[T](legacyDB, tx)
	if err != nil {
		return err
	}
	summary.SyncRows += n
	return nil
}

func countLegacySyncRows(legacyDB *gorm.DB) (int, error) {
	tables := []string{"ban_comments", "ban_illusts", "ban_tags", "ban_users", "illust_histories", "novel_histories", "tag_histories"}
	total := 0
	for _, table := range tables {
		var count int64
		if err := legacyDB.Table(table).Count(&count).Error; err != nil {
			return total, err
		}
		total += int(count)
	}
	return total, nil
}

func importLegacyBookmarks(ctx context.Context, legacyDB *gorm.DB, dryRun bool, summary *ImportLegacySummary) error {
	var illusts []model.PixezBookmarkIllust
	if err := legacyDB.Find(&illusts).Error; err != nil {
		return fmt.Errorf("read legacy bookmark_illusts: %w", err)
	}
	var novels []model.PixezBookmarkNovel
	if err := legacyDB.Find(&novels).Error; err != nil {
		return fmt.Errorf("read legacy bookmark_novels: %w", err)
	}
	summary.BookmarkIllusts = len(illusts)
	summary.BookmarkNovels = len(novels)
	if dryRun {
		return nil
	}
	for i := range illusts {
		illusts[i].ID = 0
	}
	for i := range novels {
		novels[i].ID = 0
	}
	if len(illusts) > 0 {
		if err := db.DB(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "pixiv_user_id"}, {Name: "restrict"}, {Name: "illust_id"}},
			UpdateAll: true,
		}).Create(&illusts).Error; err != nil {
			return err
		}
	}
	if len(novels) > 0 {
		if err := db.DB(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "pixiv_user_id"}, {Name: "restrict"}, {Name: "novel_id"}},
			UpdateAll: true,
		}).Create(&novels).Error; err != nil {
			return err
		}
	}
	return nil
}

func importLegacyMirrors(ctx context.Context, legacyDB *gorm.DB, mirrorDir string, dryRun bool, summary *ImportLegacySummary) error {
	var illusts []model.PixezMirrorIllust
	if err := legacyDB.Find(&illusts).Error; err != nil {
		return fmt.Errorf("read legacy mirror_illust: %w", err)
	}
	var novels []model.PixezMirrorNovel
	if err := legacyDB.Find(&novels).Error; err != nil {
		return fmt.Errorf("read legacy mirror_novel: %w", err)
	}
	summary.MirrorIllusts = len(illusts)
	summary.MirrorNovels = len(novels)
	if dryRun {
		return countLegacyMirrorFiles(mirrorDir, illusts, summary)
	}

	for _, illust := range illusts {
		imported, missing, err := importLegacyMirrorIllust(ctx, legacyDB, mirrorDir, illust)
		summary.ImportedFiles += imported
		summary.MissingFiles += missing
		if err != nil {
			return err
		}
	}
	for i := range novels {
		novels[i] = prepareLegacyMirrorNovel(legacyDB, novels[i])
	}
	if len(novels) > 0 {
		if err := db.DB(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "novel_id"}},
			UpdateAll: true,
		}).Create(&novels).Error; err != nil {
			return err
		}
	}
	return nil
}

func prepareLegacyMirrorNovel(legacyDB *gorm.DB, novel model.PixezMirrorNovel) model.PixezMirrorNovel {
	task := legacyMirrorTaskForTarget(legacyDB, model.PixezMirrorTargetNovel, novel.NovelID)
	novel.TaskID = task.ID
	novel.Status = legacyMirrorNovelStatus(novel, task.Status)
	novel.RequestURLsJSON = firstNonEmpty(task.RequestURLsJSON, "[]")
	novel.RetryURLsJSON = firstNonEmpty(task.RetryURLsJSON, "[]")
	novel.ErrorMessage = task.ErrorMessage
	novel.TotalCount = max(task.TotalCount, 1)
	if novel.Status == model.PixezMirrorStatusSuccess {
		novel.SuccessCount = max(task.SuccessCount, 1)
	} else {
		novel.FailedCount = max(task.FailedCount, 1)
	}
	if novel.CreatedAt.IsZero() {
		novel.CreatedAt = time.Now()
	}
	novel.UpdatedAt = time.Now()
	return novel
}

func legacyMirrorNovelStatus(novel model.PixezMirrorNovel, taskStatus string) string {
	if taskStatus != "" {
		return taskStatus
	}
	if novel.DetailJSON != "" && novel.TextJSON != "" {
		return model.PixezMirrorStatusSuccess
	}
	return model.PixezMirrorStatusFailed
}

func countLegacyMirrorFiles(mirrorDir string, illusts []model.PixezMirrorIllust, summary *ImportLegacySummary) error {
	for _, illust := range illusts {
		var files []legacyImageFileRecord
		if err := json.Unmarshal([]byte(illust.ImageFilesJSON), &files); err != nil {
			continue
		}
		for _, file := range files {
			if _, err := os.Stat(resolveLegacyMirrorPath(mirrorDir, file)); err == nil {
				summary.ImportedFiles++
			} else {
				summary.MissingFiles++
			}
		}
	}
	return nil
}

func importLegacyMirrorIllust(ctx context.Context, legacyDB *gorm.DB, mirrorDir string, old model.PixezMirrorIllust) (int, int, error) {
	var oldFiles []legacyImageFileRecord
	if old.ImageFilesJSON != "" {
		_ = json.Unmarshal([]byte(old.ImageFilesJSON), &oldFiles)
	}
	newFiles := make([]model.PixezMirrorImageFile, 0, len(oldFiles))
	requestURLs := make([]string, 0, len(oldFiles))
	imported := 0
	missing := 0
	for idx, file := range oldFiles {
		if file.URL != "" {
			requestURLs = append(requestURLs, file.URL)
		}
		filePath := resolveLegacyMirrorPath(mirrorDir, file)
		data, err := os.ReadFile(filePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				missing++
				continue
			}
			return imported, missing, err
		}
		mimeType := http.DetectContentType(data[:min(len(data), detectContentBytes)])
		if ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(filePath)), "."); ext != "" {
			if byExt := mime.TypeByExtension("." + ext); byExt != "" {
				mimeType = byExt
			}
		}
		record, err := registerMirrorUpload(ctx, file.URL, idx, data, mimeType)
		if err != nil {
			return imported, missing, err
		}
		if record.FileName == "" && file.Filename != "" {
			record.FileName = file.Filename
		}
		newFiles = append(newFiles, record)
		imported++
	}

	task := legacyMirrorTaskForTarget(legacyDB, model.PixezMirrorTargetIllust, old.IllustID)
	status := task.Status
	if status == "" {
		status = model.PixezMirrorStatusFailed
		if len(newFiles) > 0 {
			status = model.PixezMirrorStatusSuccess
		}
	}
	record := old
	record.TaskID = task.ID
	record.Status = status
	record.ImageFilesJSON = mustJSON(newFiles)
	record.RequestURLsJSON = firstNonEmpty(task.RequestURLsJSON, mustJSON(requestURLs))
	record.RetryURLsJSON = firstNonEmpty(task.RetryURLsJSON, "[]")
	record.ErrorMessage = task.ErrorMessage
	record.TotalCount = max(task.TotalCount, len(oldFiles))
	record.SuccessCount = max(task.SuccessCount, len(newFiles))
	record.FailedCount = max(task.FailedCount, missing)
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}
	record.UpdatedAt = time.Now()

	err := db.DB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "illust_id"}},
		UpdateAll: true,
	}).Create(&record).Error
	return imported, missing, err
}

func legacyMirrorTaskForTarget(legacyDB *gorm.DB, targetType string, targetID int64) legacyMirrorTask {
	var task legacyMirrorTask
	_ = legacyDB.Where("target_type = ? AND target_id = ?", targetType, targetID).First(&task).Error
	return task
}

func resolveLegacyMirrorPath(mirrorDir string, file legacyImageFileRecord) string {
	if file.Path != "" {
		if filepath.IsAbs(file.Path) {
			return file.Path
		}
		if strings.HasPrefix(filepath.ToSlash(file.Path), "data/mirror/") {
			return filepath.Join(filepath.Dir(mirrorDir), strings.TrimPrefix(filepath.ToSlash(file.Path), "data/"))
		}
		return file.Path
	}
	if file.Filename == "" {
		return ""
	}
	illustID, ok := leadingNumericID(file.Filename)
	if !ok {
		return filepath.Join(mirrorDir, file.Filename)
	}
	return filepath.Join(mirrorDir, fmt.Sprintf("%d", illustID), file.Filename)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func fallbackSiblingPath(pathValue string) string {
	if pathValue == "" || filepath.IsAbs(pathValue) {
		return pathValue
	}
	if _, err := os.Stat(pathValue); err == nil {
		return pathValue
	}
	sibling := filepath.Join("..", pathValue)
	if _, err := os.Stat(sibling); err == nil {
		return sibling
	}
	return pathValue
}
