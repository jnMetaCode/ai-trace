package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ai-trace/server/internal/cert"
	"github.com/gin-gonic/gin"
)

// SearchCerts 搜索存证
// @Summary 搜索存证证书
// @Description 获取当前租户的存证证书列表
// @Tags Certificates
// @Produce json
// @Param X-API-Key header string true "AI-Trace 平台 API Key"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量（最大100）" default(20)
// @Success 200 {object} map[string]interface{} "证书列表"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Security ApiKeyAuth
// @Router /certs/search [get]
func (h *Handler) SearchCerts(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	// 解析分页参数
	page := 1
	pageSize := 20
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if sizeStr := c.Query("page_size"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= 100 {
			pageSize = s
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// 获取总数（用于分页）
	var totalCount int
	err := h.stores.DB.QueryRow(ctx, `
		SELECT COUNT(*) FROM certificates WHERE tenant_id = $1
	`, tenantID).Scan(&totalCount)
	if err != nil {
		h.logger.Warnf("Failed to count certificates: %v", err)
		totalCount = 0
	}

	rows, err := h.stores.DB.Query(ctx, `
		SELECT cert_id, trace_id, root_hash, event_count, evidence_level, created_at
		FROM certificates
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, tenantID, pageSize, (page-1)*pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Query failed",
		})
		return
	}
	defer rows.Close()

	certs := make([]map[string]interface{}, 0)
	for rows.Next() {
		var certID, traceID, rootHash, evidenceLevel string
		var eventCount int
		var createdAt time.Time

		if err := rows.Scan(&certID, &traceID, &rootHash, &eventCount, &evidenceLevel, &createdAt); err != nil {
			h.logger.Warnf("Failed to scan certificate row: %v", err)
			continue
		}

		certs = append(certs, map[string]interface{}{
			"cert_id":        certID,
			"trace_id":       traceID,
			"root_hash":      rootHash,
			"event_count":    eventCount,
			"evidence_level": evidenceLevel,
			"created_at":     createdAt,
		})
	}

	// 检查行迭代错误
	if err := rows.Err(); err != nil {
		h.logger.Errorf("Error iterating certificate rows: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read certificates",
		})
		return
	}

	// 计算分页信息
	totalPages := (totalCount + pageSize - 1) / pageSize
	hasMore := page < totalPages

	c.JSON(http.StatusOK, gin.H{
		"certificates": certs,
		"pagination": map[string]interface{}{
			"page":        page,
			"page_size":   pageSize,
			"total_count": totalCount,
			"total_pages": totalPages,
			"has_more":    hasMore,
		},
		"size": len(certs),
	})
}

// GetCert 获取存证详情
// @Summary 获取存证证书详情
// @Description 根据证书 ID 获取存证证书的完整信息
// @Tags Certificates
// @Produce json
// @Param X-API-Key header string true "AI-Trace 平台 API Key"
// @Param cert_id path string true "证书 ID"
// @Success 200 {object} cert.Certificate "证书详情"
// @Failure 404 {object} map[string]interface{} "证书不存在"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Security ApiKeyAuth
// @Router /certs/{cert_id} [get]
func (h *Handler) GetCert(c *gin.Context) {
	certID := c.Param("cert_id")
	tenantID := c.GetString("tenant_id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var certData []byte
	err := h.stores.DB.QueryRow(ctx, `
		SELECT cert_data FROM certificates WHERE cert_id = $1 AND tenant_id = $2
	`, certID, tenantID).Scan(&certData)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Certificate not found",
			"message": fmt.Sprintf("No certificate found with ID '%s'.", certID),
			"suggestions": []string{
				"Verify the cert_id is correct (should start with 'cert_')",
				"Use GET /api/v1/certs/search to list available certificates",
				"Generate a new certificate using POST /api/v1/certs/commit",
			},
		})
		return
	}

	var certificate cert.Certificate
	if err := json.Unmarshal(certData, &certificate); err != nil {
		h.logger.Errorf("Failed to unmarshal certificate: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse certificate data",
		})
		return
	}

	c.JSON(http.StatusOK, certificate)
}
