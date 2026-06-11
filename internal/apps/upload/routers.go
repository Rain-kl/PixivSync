// Copyright 2025 linux.do
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package upload

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Rain-kl/Wavelet/internal/apps/oauth"
	"github.com/Rain-kl/Wavelet/internal/config"
	"github.com/Rain-kl/Wavelet/internal/db"
	"github.com/Rain-kl/Wavelet/internal/db/idgen"
	"github.com/Rain-kl/Wavelet/internal/logger"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/storage"
	"github.com/Rain-kl/Wavelet/internal/util"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type batchDownloadRequest struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// UploadFile 通用上传文件接口
// @Summary 上传文件
// @Description 支持各种类型的通用文件上传，支持自动文件类型检测、哈希计算与“秒传”去重
// @Tags upload
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "要上传的文件"
// @Param type formData string false "业务分类 (例如: avatar, attachment, doc，默认为 generic)"
// @Param metadata formData string false "额外的 JSON 格式元数据"
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny{data=model.Upload} "上传成功"
// @Failure 400 {object} util.ResponseAny "请求参数错误或文件受限"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 500 {object} util.ResponseAny "内部错误"
// @Router /api/v1/upload [post]
//
//nolint:revive
func UploadFile(c *gin.Context) {
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "sandbox")

	// 限制请求体大小以防止 DoS
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	currUser, _ := util.GetFromContext[*model.User](c, oauth.UserObjKey)
	ctx := c.Request.Context()

	header, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, util.Err(ErrNoFileSelected))
		return
	}

	file, err := header.Open()
	if err != nil {
		c.JSON(http.StatusOK, util.Err(ErrOpenFileFailed))
		return
	}
	defer func() { _ = file.Close() }()

	// 校验大小
	if header.Size > maxUploadSize {
		c.JSON(http.StatusOK, util.Err(ErrGenericFileTooLarge))
		return
	}

	// 2. 提取文件基本元数据
	origName := header.Filename
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(origName), "."))
	if ext == "" {
		ext = "bin"
	}

	// 3. 校验文件后缀是否在允许的系统配置列表中
	if errMsg := validateUploadExtension(ctx, ext); errMsg != "" {
		c.JSON(http.StatusOK, util.Err(errMsg))
		return
	}

	// 4. 读取文件并计算 Hash
	hashWriter := sha256.New()
	var buf bytes.Buffer
	size, err := io.Copy(&buf, io.TeeReader(file, hashWriter))
	if err != nil {
		c.JSON(http.StatusOK, util.Err(ErrProcessFileFailed))
		return
	}

	fileHash := hex.EncodeToString(hashWriter.Sum(nil))
	mimeType := detectMimeType(&buf, header, size)

	// 校验真实 MIME Type 是否与常见图片扩展名匹配，防止 Polyglot / HTML 注入攻击
	if isImageExtension(ext) && !strings.HasPrefix(mimeType, "image/") {
		c.JSON(http.StatusOK, util.Err(ErrFileContentExtensionMismatch))
		return
	}

	// 6. 秒传匹配校验：校验数据库中是否存在相同 Hash 且大小一致的可用文件
	handled, lookupErr := tryInstantUpload(ctx, c, currUser, fileHash, size, mimeType, ext, origName)
	if handled {
		return
	}
	if lookupErr != nil && !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusOK, util.Err(ErrFileValidationFailed))
		return
	}

	// 7. 解析可选元数据字段
	meta, errMsg := parseUploadMetadata(c, mimeType)
	if errMsg != "" {
		c.JSON(http.StatusOK, util.Err(errMsg))
		return
	}

	id := idgen.NextUint64ID()
	subPath := fmt.Sprintf("uploads/%s/%d.%s", time.Now().Format("2006/01/02"), id, ext)

	// 8. 写入底层存储驱动 (优先 S3 驱动，无配置或未开启则 fallback 至本地文件)
	storageDriver, subPath, errMsg := storeUploadFile(ctx, id, ext, subPath, size, mimeType, &buf, &meta)
	if errMsg != "" {
		c.JSON(http.StatusOK, util.Err(errMsg))
		return
	}

	// 9. 保存文件记录至数据库
	newUpload := model.Upload{
		ID:            id,
		UserID:        currUser.ID,
		FileName:      origName,
		FilePath:      subPath,
		FileSize:      size,
		MimeType:      mimeType,
		Extension:     ext,
		Hash:          fileHash,
		StorageDriver: storageDriver,
		Type:          c.DefaultPostForm("type", "generic"),
		Status:        model.UploadStatusUsed,
		Metadata:      meta,
	}

	if err := saveUploadRecord(ctx, &newUpload, storageDriver, subPath); err != "" {
		c.JSON(http.StatusOK, util.Err(err))
		return
	}

	c.JSON(http.StatusOK, util.OK(newUpload))
}

// DownloadFile 通用单文件下载接口
// @Summary 下载单文件
// @Description 根据文件 ID 获取文件，以附件形式 (Attachment) 强制开启客户端浏览器下载
// @Tags upload
// @Produce octet-stream
// @Param id path string true "文件 ID"
// @Param compress query string false "是否启用压缩 (传任意非空值代表启用，非图片文件将被忽略)"
// @Param level query string false "压缩质量等级 (low, medium, high)，默认为 high"
// @Security SessionCookie
// @Success 200 {file} file "成功下载文件"
// @Failure 400 {object} util.ResponseAny "参数错误"
// @Failure 404 {object} util.ResponseAny "文件不存在"
// @Failure 500 {object} util.ResponseAny "服务内部错误"
// @Router /api/v1/upload/download/{id} [get]
func DownloadFile(c *gin.Context) {
	upload, err := getUploadRecordByID(c)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		if _, ok := err.(*strconv.NumError); ok {
			c.JSON(http.StatusOK, util.Err(ErrInvalidFileID))
			return
		}
		c.JSON(http.StatusOK, util.Err(ErrQueryUploadRecordFailed))
		return
	}

	fileName := upload.FileName
	compressStr := c.Query("compress")
	isImage := strings.HasPrefix(strings.ToLower(upload.MimeType), "image/") || isImageExtension(strings.ToLower(upload.Extension))

	if compressStr != "" && isImage {
		ext := filepath.Ext(fileName)
		if ext != "" {
			fileName = strings.TrimSuffix(fileName, ext) + ".webp"
		} else {
			fileName += ".webp"
		}
	}

	// 设置下载 Attachment 响应头 (支持 UTF-8 中文文件名转义)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(fileName)))
	ServeUpload(c, upload)
}

// BatchDownloadFiles 批量打包 ZIP 下载接口
// @Summary 批量打包下载
// @Description 传入多个文件 ID，后台实时将其打包压缩为 ZIP 流并输出，自动处理文件名重复冲突
// @Tags upload
// @Accept json
// @Produce octet-stream
// @Param request body upload.batchDownloadRequest true "包含文件 ID 数组的请求体"
// @Security SessionCookie
// @Success 200 {file} file "成功下载打包后的 ZIP"
// @Failure 400 {object} util.ResponseAny "参数错误"
// @Failure 500 {object} util.ResponseAny "打包失败"
// @Router /api/v1/upload/download/batch [post]
func BatchDownloadFiles(c *gin.Context) {
	ctx := c.Request.Context()

	var req batchDownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, util.Err(ErrInvalidBatchDownloadRequest))
		return
	}

	// 转换 ID 列表
	var ids []uint64
	for _, idStr := range req.IDs {
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusOK, util.Err(fmt.Sprintf(ErrInvalidIDValueFormat, idStr)))
			return
		}
		ids = append(ids, id)
	}

	// 查库获取所有匹配且正常的文件记录
	var uploads []model.Upload
	if err := db.DB(ctx).Where("id IN ? AND status IN (?, ?)", ids, model.UploadStatusPending, model.UploadStatusUsed).Find(&uploads).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(ErrRetrieveUploadRecordsFailed))
		return
	}

	if len(uploads) == 0 {
		c.JSON(http.StatusOK, util.Err(ErrNoValidFilesForArchive))
		return
	}

	// 设置 ZIP 格式流的响应头
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename=\"batch_download.zip\"")

	// 开启实时 ZIP 压缩器并直接输出给 Response Writer
	zipWriter := zip.NewWriter(c.Writer)
	defer func() { _ = zipWriter.Close() }()

	// 用于解决 ZIP 内部文件名称发生碰撞冲突的问题
	usedNames := make(map[string]int)

	for _, upload := range uploads {
		// 校验防冲突重命名逻辑
		fileName := upload.FileName
		if count, exists := usedNames[fileName]; exists {
			usedNames[fileName] = count + 1
			ext := filepath.Ext(fileName)
			base := strings.TrimSuffix(fileName, ext)
			fileName = fmt.Sprintf("%s_%d%s", base, count, ext)
		} else {
			usedNames[fileName] = 1
		}

		// 在 ZIP 包内建新条目
		zipFileEntry, err := zipWriter.Create(fileName)
		if err != nil {
			logger.ErrorF(ctx, "ZIP 添加条目失败 [%s]: %v", fileName, err)
			continue
		}

		// 打开底层文件数据源
		var rc io.ReadCloser
		if upload.StorageDriver == "local" || (upload.StorageDriver == "" && !storage.IsEnabled()) {
			fileSrc, err := os.Open(upload.FilePath)
			if err != nil {
				logger.ErrorF(ctx, "打包时读取本地文件失败: %v", err)
				continue
			}
			rc = fileSrc
		} else {
			obj, err := storage.GetObject(ctx, upload.FilePath)
			if err != nil {
				logger.ErrorF(ctx, "打包时拉取 S3 文件失败: %v", err)
				continue
			}
			rc = obj.Body
		}

		// 流式拷贝到 ZIP entry
		_, err = io.Copy(zipFileEntry, rc)
		_ = rc.Close()
		if err != nil {
			logger.ErrorF(ctx, "写入 ZIP 流失败: %v", err)
		}
	}
}

type listMyFilesRequest struct {
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
	Keyword   string `form:"keyword"`
	Type      string `form:"type"`
	Extension string `form:"extension"`
}

type listMyFilesResponse struct {
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
	Items    []model.Upload `json:"items"`
}

// ListMyFiles 获取当前用户上传的文件列表
// @Summary 获取我的文件列表
// @Description 分页获取当前登录用户上传的文件，支持文件名关键词、业务类型、扩展名过滤
// @Tags upload
// @Produce json
// @Param page query int false "页码（默认 1）"
// @Param page_size query int false "每页数量（默认 20，最大 100）"
// @Param keyword query string false "文件名关键词（模糊匹配）"
// @Param type query string false "业务分类过滤"
// @Param extension query string false "扩展名过滤"
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny{data=listMyFilesResponse} "查询成功"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Router /api/v1/upload/my [get]
func ListMyFiles(c *gin.Context) {
	currUser, _ := util.GetFromContext[*model.User](c, oauth.UserObjKey)
	ctx := c.Request.Context()

	var req listMyFilesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, util.Err(ErrInvalidParams))
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	query := db.DB(ctx).Model(&model.Upload{}).
		Where("user_id = ? AND status != ?", currUser.ID, model.UploadStatusDeleted)

	if req.Keyword != "" {
		query = query.Where("file_name ILIKE ?", "%"+req.Keyword+"%")
	}
	if req.Type != "" {
		query = query.Where("type = ?", req.Type)
	}
	if req.Extension != "" {
		query = query.Where("extension = ?", strings.ToLower(req.Extension))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(ErrQueryFileCountFailed))
		return
	}

	var items []model.Upload
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(req.PageSize).Find(&items).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(ErrQueryFileListFailed))
		return
	}

	c.JSON(http.StatusOK, util.OK(listMyFilesResponse{
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Items:    items,
	}))
}

// DeleteFile 软删除文件记录
// @Summary 删除文件
// @Description 将文件状态置为 deleted（软删除），不会立即清理底层存储对象
// @Tags upload
// @Produce json
// @Param id path string true "文件 ID"
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny "删除成功"
// @Failure 403 {object} util.ResponseAny "无权操作"
// @Failure 404 {object} util.ResponseAny "文件不存在"
// @Router /api/v1/upload/{id} [delete]
func DeleteFile(c *gin.Context) {
	currUser, _ := util.GetFromContext[*model.User](c, oauth.UserObjKey)
	ctx := c.Request.Context()

	idStr := c.Param("id")
	uploadID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, util.Err(ErrInvalidFileID))
		return
	}

	var upload model.Upload
	if err := db.DB(ctx).Where("id = ? AND status != ?", uploadID, model.UploadStatusDeleted).First(&upload).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.JSON(http.StatusOK, util.Err(ErrQueryUploadRecordFailed))
		return
	}

	// 仅允许文件所有者或管理员删除
	if upload.UserID != currUser.ID && !currUser.IsAdmin {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	if err := db.DB(ctx).Model(&upload).Update("status", model.UploadStatusDeleted).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(ErrDeleteFileFailed))
		return
	}

	c.JSON(http.StatusOK, util.OKNil())
}

// validateUploadExtension 校验文件后缀是否在系统允许的上传扩展名列表中
func validateUploadExtension(ctx context.Context, ext string) string {
	var sc model.SystemConfig
	if err := sc.GetByKey(ctx, model.ConfigKeyUploadAllowedExtensions); err == nil && sc.Value != "" {
		allowedExts := strings.Split(strings.ToLower(sc.Value), ",")
		allowed := false
		for _, allowedExt := range allowedExts {
			if strings.TrimSpace(allowedExt) == ext {
				allowed = true
				break
			}
		}
		if !allowed {
			return ErrUnsupportedFormat
		}
	}
	return ""
}

// tryInstantUpload 尝试秒传：若数据库已存在相同 Hash 且大小一致的可用文件，直接生成新记录
func tryInstantUpload(ctx context.Context, c *gin.Context, currUser *model.User, fileHash string, size int64, mimeType, ext, origName string) (bool, error) {
	var existing model.Upload
	err := db.DB(ctx).Where("hash = ? AND file_size = ? AND status IN (?, ?)", fileHash, size, model.UploadStatusPending, model.UploadStatusUsed).First(&existing).Error
	if err != nil {
		return false, err
	}

	id := idgen.NextUint64ID()
	newUpload := model.Upload{
		ID:            id,
		UserID:        currUser.ID,
		FileName:      origName,
		FilePath:      existing.FilePath,
		FileSize:      size,
		MimeType:      mimeType,
		Extension:     ext,
		Hash:          fileHash,
		StorageDriver: existing.StorageDriver,
		Type:          c.DefaultPostForm("type", "generic"),
		Status:        model.UploadStatusUsed,
		Metadata:      existing.Metadata,
	}

	if err := db.DB(ctx).Create(&newUpload).Error; err != nil {
		c.JSON(http.StatusOK, util.Err(ErrSaveUploadRecordFailed))
		return true, err
	}

	logger.InfoF(ctx, "文件触发秒传成功! ID: %d, Path: %s", id, existing.FilePath)
	c.JSON(http.StatusOK, util.OK(newUpload))
	return true, nil
}

// storeUploadFile 将文件写入底层存储驱动（S3 或本地磁盘）
func storeUploadFile(ctx context.Context, id uint64, ext, subPath string, size int64, mimeType string, buf *bytes.Buffer, meta *model.UploadMetadata) (string, string, string) {
	if storage.IsEnabled() {
		meta.Bucket = config.Config.S3.Bucket
		fullKey := storage.BuildKey(subPath)
		if err := storage.PutObject(ctx, fullKey, bytes.NewReader(buf.Bytes()), size, mimeType); err != nil {
			logger.ErrorF(ctx, "S3 存储上传失败: %v", err)
			return "", "", ErrSaveFileFailed
		}
		return "s3", subPath, ""
	}

	localDir := filepath.Join("uploads", time.Now().Format("2006/01/02"))
	if err := os.MkdirAll(localDir, uploadDirPerm); err != nil {
		logger.ErrorF(ctx, "创建本地上传目录失败: %v", err)
		return "", "", ErrSaveFileFailed
	}

	localPath := filepath.Join(localDir, fmt.Sprintf("%d.%s", id, ext))
	if err := os.WriteFile(localPath, buf.Bytes(), uploadFilePerm); err != nil {
		logger.ErrorF(ctx, "本地磁盘写入文件失败: %v", err)
		return "", "", ErrSaveFileFailed
	}
	return "local", localPath, ""
}

// isImageExtension 判断文件扩展名是否属于常见图片格式
func isImageExtension(ext string) bool {
	for _, imgExt := range []string{"jpg", "jpeg", "png", "webp", "gif"} {
		if ext == imgExt {
			return true
		}
	}
	return false
}

// parseUploadMetadata 解析上传元数据字段
func parseUploadMetadata(c *gin.Context, mimeType string) (model.UploadMetadata, string) {
	var meta model.UploadMetadata
	metadataStr := c.DefaultPostForm("metadata", "")
	if metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &meta); err != nil {
			return meta, ErrInvalidMetadataJSON
		}
	}
	meta.OriginalMime = mimeType
	meta.UserAgent = c.Request.UserAgent()
	meta.ClientIP = c.ClientIP()
	return meta, ""
}

// detectMimeType 检测文件的 MIME 类型，优先使用 Content-Type 头部信息
func detectMimeType(buf *bytes.Buffer, header *multipart.FileHeader, size int64) string {
	mimeType := http.DetectContentType(buf.Bytes()[:min(detectContentBytes, int(size))])
	if mimeType == "application/octet-stream" && header.Header.Get("Content-Type") != "" {
		mimeType = header.Header.Get("Content-Type")
	}
	return mimeType
}

// saveUploadRecord 保存上传记录到数据库，失败时清理本地垃圾文件
func saveUploadRecord(ctx context.Context, upload *model.Upload, storageDriver, filePath string) string {
	if err := db.DB(ctx).Create(upload).Error; err != nil {
		if storageDriver == "local" {
			_ = os.Remove(filePath)
		}
		return ErrSaveUploadRecordFailed
	}
	return ""
}

// GetDistinctUploadTypes 获取数据库中所有已存在的文件业务类型
// @Summary 获取文件业务类型列表
// @Description 返回数据库中所有已上传文件实际拥有的业务类型列表
// @Tags admin
// @Produce json
// @Security SessionCookie
// @Success 200 {object} util.ResponseAny{data=[]string} "业务类型列表"
// @Failure 401 {object} util.ResponseAny "未登录"
// @Failure 403 {object} util.ResponseAny "无管理员权限"
// @Failure 500 {object} util.ResponseAny "内部错误"
// @Router /api/v1/admin/uploads/types [get]
func GetDistinctUploadTypes(c *gin.Context) {
	var dbTypes []string
	if err := db.DB(c.Request.Context()).Model(&model.Upload{}).
		Where("type IS NOT NULL AND type != ''").
		Distinct().
		Pluck("type", &dbTypes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, util.Err(err.Error()))
		return
	}

	sort.Strings(dbTypes)
	c.JSON(http.StatusOK, util.OK(dbTypes))
}
