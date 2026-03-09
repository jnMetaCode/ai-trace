package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ai-trace/server/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DecryptContent 解密加密内容
// @Summary 解密内容
// @Description 解密已加密存储的 prompt 或 output 内容
// @Description
// @Description ## 权限要求
// @Description - 只有内容所有者或授权方可解密
// @Description - 所有解密操作会被记录到审计日志
// @Tags Decrypt
// @Accept json
// @Produce json
// @Param X-API-Key header string true "AI-Trace 平台 API Key"
// @Param request body DecryptRequest true "解密请求"
// @Success 200 {object} DecryptResponse "解密结果"
// @Failure 400 {object} map[string]interface{} "请求无效"
// @Failure 403 {object} map[string]interface{} "无权限"
// @Failure 404 {object} map[string]interface{} "内容未找到"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Security ApiKeyAuth
// @Router /decrypt [post]
func (h *Handler) DecryptContent(c *gin.Context) {
	var req DecryptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// 获取当前用户信息
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	// 创建审计存储
	auditStore := store.NewDecryptAuditStore(h.stores.DB)
	auditLogID := fmt.Sprintf("audit_%s", uuid.New().String()[:8])
	decryptedAt := time.Now()

	// 创建审计记录（用于成功或失败的日志）
	auditRecord := &store.DecryptAuditRecord{
		AuditLogID:   auditLogID,
		EncryptedRef: req.EncryptedRef,
		ContentType:  req.ContentType,
		TenantID:     tenantID,
		UserID:       userID,
		ClientIP:     c.ClientIP(),
		UserAgent:    c.GetHeader("User-Agent"),
		TraceID:      req.TraceID,
		DecryptedAt:  decryptedAt,
	}

	// 验证权限
	if !h.canDecrypt(c, req.EncryptedRef, tenantID, userID) {
		auditRecord.Success = false
		auditRecord.FailReason = "permission_denied"
		if err := auditStore.Log(c.Request.Context(), auditRecord); err != nil {
			h.logger.Warnw("Failed to log decrypt audit", "error", err)
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error": "You don't have permission to decrypt this content",
		})
		return
	}

	// 检查加密存储是否可用
	if h.encryptedStore == nil {
		auditRecord.Success = false
		auditRecord.FailReason = "encryption_not_configured"
		if err := auditStore.Log(c.Request.Context(), auditRecord); err != nil {
			h.logger.Warnw("Failed to log decrypt audit", "error", err)
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Encryption service not configured",
		})
		return
	}

	// 解密内容
	plaintext, err := h.encryptedStore.RetrieveByRef(c.Request.Context(), tenantID, req.EncryptedRef)
	if err != nil {
		auditRecord.Success = false
		if err == store.ErrContentNotFound {
			auditRecord.FailReason = "content_not_found"
			if logErr := auditStore.Log(c.Request.Context(), auditRecord); logErr != nil {
				h.logger.Warnw("Failed to log decrypt audit", "error", logErr)
			}
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Encrypted content not found",
				"message": "The specified encrypted reference does not exist or has expired.",
				"suggestions": []string{
					"Verify the encrypted_ref value is correct",
					"Check if the content was stored with encryption enabled",
					"Encrypted content may have a retention policy",
				},
			})
			return
		}

		if err == store.ErrDecryptionFailed {
			auditRecord.FailReason = "decryption_failed"
			if logErr := auditStore.Log(c.Request.Context(), auditRecord); logErr != nil {
				h.logger.Warnw("Failed to log decrypt audit", "error", logErr)
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to decrypt content",
			})
			return
		}

		auditRecord.FailReason = "internal_error"
		if logErr := auditStore.Log(c.Request.Context(), auditRecord); logErr != nil {
			h.logger.Warnw("Failed to log decrypt audit", "error", logErr)
		}
		h.logger.Errorw("Failed to retrieve encrypted content", "error", err, "ref", req.EncryptedRef)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve content",
		})
		return
	}

	// 记录成功的审计日志
	auditRecord.Success = true
	if err := auditStore.Log(c.Request.Context(), auditRecord); err != nil {
		h.logger.Warnw("Failed to log decrypt audit", "error", err)
	}

	// 返回解密后的内容
	c.JSON(http.StatusOK, DecryptResponse{
		Content:     string(plaintext),
		ContentType: req.ContentType,
		DecryptedAt: decryptedAt,
		AuditLogID:  auditLogID,
	})

	h.logger.Infow("Content decrypted successfully",
		"audit_log_id", auditLogID,
		"encrypted_ref", req.EncryptedRef,
		"tenant_id", tenantID,
		"user_id", userID,
	)
}

// canDecrypt 检查用户是否有权解密
func (h *Handler) canDecrypt(c *gin.Context, encryptedRef, tenantID, userID string) bool {
	// 必须有租户 ID
	if tenantID == "" {
		return false
	}

	// encryptedRef 至少要包含 tenant_id
	if len(encryptedRef) < len(tenantID) {
		return false
	}

	// 验证 encrypted_ref 属于当前租户
	// ref 格式: minio://bucket/encrypted/{tenant_id}/{content_type}/{uuid}
	// 严格检查 ref 前缀是否匹配租户 ID
	expectedPrefix := fmt.Sprintf("minio://%s/encrypted/%s/", h.config.Minio.Bucket, tenantID)
	if len(encryptedRef) >= len(expectedPrefix) && encryptedRef[:len(expectedPrefix)] == expectedPrefix {
		return true
	}

	// 备选格式验证 (不同的 bucket 配置)
	altPrefix := fmt.Sprintf("encrypted/%s/", tenantID)
	if strings.Contains(encryptedRef, altPrefix) {
		// 确保 tenant_id 在正确的位置（在 "encrypted/" 之后）
		idx := strings.Index(encryptedRef, "encrypted/")
		if idx >= 0 {
			afterEncrypted := encryptedRef[idx+len("encrypted/"):]
			if strings.HasPrefix(afterEncrypted, tenantID+"/") {
				return true
			}
		}
	}

	return false
}

// GetDecryptAuditLogs 获取解密审计日志
// @Summary 获取解密审计日志
// @Description 获取指定内容的解密审计历史
// @Tags Decrypt
// @Produce json
// @Param encrypted_ref query string true "加密内容引用"
// @Param limit query int false "返回数量限制" default(20)
// @Param offset query int false "偏移量" default(0)
// @Success 200 {object} DecryptAuditLogsResponse "审计日志"
// @Failure 403 {object} map[string]interface{} "无权限"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Security ApiKeyAuth
// @Router /decrypt/audit [get]
func (h *Handler) GetDecryptAuditLogs(c *gin.Context) {
	encryptedRef := c.Query("encrypted_ref")
	if encryptedRef == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "encrypted_ref is required",
		})
		return
	}

	tenantID := c.GetString("tenant_id")

	// 解析分页参数
	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// 验证权限：只有属于该租户的内容才能查看审计日志
	if !h.canDecrypt(c, encryptedRef, tenantID, "") {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "You don't have permission to view audit logs for this content",
		})
		return
	}

	// 从数据库查询审计日志
	auditStore := store.NewDecryptAuditStore(h.stores.DB)
	records, total, err := auditStore.GetByEncryptedRef(c.Request.Context(), tenantID, encryptedRef, limit, offset)
	if err != nil {
		h.logger.Errorw("Failed to get decrypt audit logs", "error", err, "encrypted_ref", encryptedRef)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve audit logs",
		})
		return
	}

	// 转换为响应格式
	logs := make([]DecryptAuditLog, 0, len(records))
	for _, r := range records {
		logs = append(logs, DecryptAuditLog{
			AuditLogID:  r.AuditLogID,
			UserID:      r.UserID,
			TenantID:    r.TenantID,
			DecryptedAt: r.DecryptedAt,
			ClientIP:    r.ClientIP,
			Success:     r.Success,
			FailReason:  r.FailReason,
		})
	}

	c.JSON(http.StatusOK, DecryptAuditLogsResponse{
		EncryptedRef: encryptedRef,
		Logs:         logs,
		TotalCount:   int(total),
	})
}

// DecryptRequest 解密请求
type DecryptRequest struct {
	EncryptedRef string `json:"encrypted_ref" binding:"required"` // 加密内容引用
	ContentType  string `json:"content_type" binding:"required"`  // prompt 或 output
	TraceID      string `json:"trace_id,omitempty"`               // 可选的 trace_id
}

// DecryptResponse 解密响应
type DecryptResponse struct {
	Content     string    `json:"content"`      // 解密后的内容
	ContentType string    `json:"content_type"` // 内容类型
	DecryptedAt time.Time `json:"decrypted_at"` // 解密时间
	AuditLogID  string    `json:"audit_log_id"` // 审计日志 ID
}

// DecryptAuditLog 解密审计日志条目
type DecryptAuditLog struct {
	AuditLogID  string    `json:"audit_log_id"`
	UserID      string    `json:"user_id"`
	TenantID    string    `json:"tenant_id"`
	DecryptedAt time.Time `json:"decrypted_at"`
	ClientIP    string    `json:"client_ip"`
	Success     bool      `json:"success"`
	FailReason  string    `json:"fail_reason,omitempty"`
}

// DecryptAuditLogsResponse 审计日志响应
type DecryptAuditLogsResponse struct {
	EncryptedRef string            `json:"encrypted_ref"`
	Logs         []DecryptAuditLog `json:"logs"`
	TotalCount   int               `json:"total_count"`
}
