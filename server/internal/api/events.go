package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ai-trace/server/internal/event"
	"github.com/gin-gonic/gin"
)

// IngestEventsRequest 事件写入请求
// @Description 批量写入事件的请求体
type IngestEventsRequest struct {
	Events []event.Event `json:"events"` // 事件列表
}

// IngestEventsResponse 事件写入响应
// @Description 事件写入的响应结果
type IngestEventsResponse struct {
	Success bool                `json:"success" example:"true"` // 是否成功
	Results []IngestEventResult `json:"results"`                // 各事件的写入结果
}

// IngestEventResult 单个事件写入结果
// @Description 单个事件的写入结果
type IngestEventResult struct {
	EventID   string `json:"event_id" example:"evt_abc123"`                              // 事件 ID
	EventHash string `json:"event_hash" example:"sha256:abcd1234..."`                   // 事件哈希
	Error     string `json:"error,omitempty" example:"duplicate event_id"`              // 错误信息（如有）
}

// IngestEvents 事件写入接口
// @Summary 批量写入事件
// @Description 将事件批量写入追踪系统，用于 SDK 直接上报场景
// @Tags Events
// @Accept json
// @Produce json
// @Param X-API-Key header string true "AI-Trace 平台 API Key"
// @Param X-Tenant-ID header string false "租户 ID"
// @Param request body IngestEventsRequest true "事件列表"
// @Success 200 {object} IngestEventsResponse "写入结果"
// @Failure 400 {object} map[string]interface{} "请求无效"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Security ApiKeyAuth
// @Router /events/ingest [post]
func (h *Handler) IngestEvents(c *gin.Context) {
	var req IngestEventsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	tenantID := c.GetString("tenant_id")
	results := make([]IngestEventResult, len(req.Events))

	for i, evt := range req.Events {
		// 设置租户ID
		if evt.TenantID == "" {
			evt.TenantID = tenantID
		}

		// 存储事件
		if err := h.storeEvent(&evt); err != nil {
			results[i] = IngestEventResult{
				EventID: evt.EventID,
				Error:   err.Error(),
			}
		} else {
			results[i] = IngestEventResult{
				EventID:   evt.EventID,
				EventHash: evt.EventHash,
			}
		}
	}

	c.JSON(http.StatusOK, IngestEventsResponse{
		Success: true,
		Results: results,
	})
}

// SearchEventsRequest 事件搜索请求
// @Description 事件搜索的查询参数
type SearchEventsRequest struct {
	TraceID   string `form:"trace_id" example:"trc_abc123"`            // 追踪 ID
	EventType string `form:"event_type" example:"INPUT"`               // 事件类型
	StartTime string `form:"start_time" example:"2024-01-01T00:00:00Z"` // 开始时间
	EndTime   string `form:"end_time" example:"2024-12-31T23:59:59Z"`   // 结束时间
	Page      int    `form:"page,default=1" example:"1"`                // 页码
	PageSize  int    `form:"page_size,default=20" example:"20"`         // 每页数量
}

// SearchEvents 事件搜索接口
// @Summary 搜索事件
// @Description 根据条件搜索追踪事件
// @Tags Events
// @Produce json
// @Param X-API-Key header string true "AI-Trace 平台 API Key"
// @Param trace_id query string false "追踪 ID"
// @Param event_type query string false "事件类型（INPUT/MODEL/OUTPUT）"
// @Param start_time query string false "开始时间（RFC3339 格式）"
// @Param end_time query string false "结束时间（RFC3339 格式）"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{} "事件列表"
// @Failure 400 {object} map[string]interface{} "请求无效"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Security ApiKeyAuth
// @Router /events/search [get]
func (h *Handler) SearchEvents(c *gin.Context) {
	var req SearchEventsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid query parameters",
		})
		return
	}

	tenantID := c.GetString("tenant_id")

	// 构建查询
	query := `
		SELECT event_id, trace_id, event_type, timestamp, payload, event_hash
		FROM events
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	argIndex := 2

	if req.TraceID != "" {
		query += fmt.Sprintf(" AND trace_id = $%d", argIndex)
		args = append(args, req.TraceID)
		argIndex++
	}

	if req.EventType != "" {
		query += fmt.Sprintf(" AND event_type = $%d", argIndex)
		args = append(args, req.EventType)
		argIndex++
	}

	// 时间范围过滤
	if req.StartTime != "" {
		if startTime, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			query += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
			args = append(args, startTime)
			argIndex++
		}
	}
	if req.EndTime != "" {
		if endTime, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			query += fmt.Sprintf(" AND timestamp <= $%d", argIndex)
			args = append(args, endTime)
			argIndex++
		}
	}

	// 构建计数查询（用于分页）
	countQuery := `SELECT COUNT(*) FROM events WHERE tenant_id = $1`
	countArgs := []interface{}{tenantID}
	countArgIndex := 2
	if req.TraceID != "" {
		countQuery += fmt.Sprintf(" AND trace_id = $%d", countArgIndex)
		countArgs = append(countArgs, req.TraceID)
		countArgIndex++
	}
	if req.EventType != "" {
		countQuery += fmt.Sprintf(" AND event_type = $%d", countArgIndex)
		countArgs = append(countArgs, req.EventType)
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var totalCount int
	if err := h.stores.DB.QueryRow(ctx, countQuery, countArgs...).Scan(&totalCount); err != nil {
		h.logger.Warnf("Failed to count events: %v", err)
		totalCount = 0
	}

	query += fmt.Sprintf(" ORDER BY timestamp DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, req.PageSize, (req.Page-1)*req.PageSize)

	rows, err := h.stores.DB.Query(ctx, query, args...)
	if err != nil {
		h.logger.Errorf("Query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Query failed",
		})
		return
	}
	defer rows.Close()

	events := make([]map[string]interface{}, 0)
	for rows.Next() {
		var eventID, traceID, eventType, eventHash string
		var timestamp time.Time
		var payload []byte

		if err := rows.Scan(&eventID, &traceID, &eventType, &timestamp, &payload, &eventHash); err != nil {
			h.logger.Warnf("Failed to scan event row: %v", err)
			continue
		}

		var payloadMap map[string]interface{}
		if err := json.Unmarshal(payload, &payloadMap); err != nil {
			h.logger.Warnf("Failed to unmarshal payload for event %s: %v", eventID, err)
			payloadMap = nil
		}

		events = append(events, map[string]interface{}{
			"event_id":   eventID,
			"trace_id":   traceID,
			"event_type": eventType,
			"timestamp":  timestamp,
			"payload":    payloadMap,
			"event_hash": eventHash,
		})
	}

	// 检查行迭代错误
	if err := rows.Err(); err != nil {
		h.logger.Errorf("Error iterating event rows: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read events",
		})
		return
	}

	// 计算分页信息
	totalPages := (totalCount + req.PageSize - 1) / req.PageSize
	hasMore := req.Page < totalPages

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"pagination": map[string]interface{}{
			"page":        req.Page,
			"page_size":   req.PageSize,
			"total_count": totalCount,
			"total_pages": totalPages,
			"has_more":    hasMore,
		},
		"size": len(events),
	})
}

// GetEvent 获取单个事件
// @Summary 获取事件详情
// @Description 根据事件 ID 获取单个事件的详细信息
// @Tags Events
// @Produce json
// @Param X-API-Key header string true "AI-Trace 平台 API Key"
// @Param event_id path string true "事件 ID"
// @Success 200 {object} map[string]interface{} "事件详情"
// @Failure 404 {object} map[string]interface{} "事件不存在"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Security ApiKeyAuth
// @Router /events/{event_id} [get]
func (h *Handler) GetEvent(c *gin.Context) {
	eventID := c.Param("event_id")
	tenantID := c.GetString("tenant_id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var traceID, eventType, eventHash string
	var timestamp time.Time
	var sequence int
	var payload []byte

	err := h.stores.DB.QueryRow(ctx, `
		SELECT trace_id, event_type, timestamp, sequence, payload, event_hash
		FROM events
		WHERE event_id = $1 AND tenant_id = $2
	`, eventID, tenantID).Scan(&traceID, &eventType, &timestamp, &sequence, &payload, &eventHash)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Event not found",
			"message": fmt.Sprintf("No event found with ID '%s'.", eventID),
			"suggestions": []string{
				"Verify the event_id is correct (should start with 'evt_')",
				"Use GET /api/v1/events/search to list available events",
				"Events are created automatically when using the chat gateway",
			},
		})
		return
	}

	var payloadMap map[string]interface{}
	if err := json.Unmarshal(payload, &payloadMap); err != nil {
		h.logger.Warnf("Failed to unmarshal payload for event %s: %v", eventID, err)
		payloadMap = nil
	}

	c.JSON(http.StatusOK, gin.H{
		"event_id":   eventID,
		"trace_id":   traceID,
		"event_type": eventType,
		"timestamp":  timestamp,
		"sequence":   sequence,
		"payload":    payloadMap,
		"event_hash": eventHash,
	})
}

// storeEvent 存储事件
func (h *Handler) storeEvent(evt *event.Event) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := h.stores.DB.Exec(ctx, `
		INSERT INTO events (
			event_id, trace_id, parent_event_id, prev_event_hash,
			event_type, timestamp, sequence,
			tenant_id, user_id, session_id,
			context, payload, payload_hash, event_hash
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
	`,
		evt.EventID, evt.TraceID, evt.ParentEventID, evt.PrevEventHash,
		evt.EventType, evt.Timestamp, evt.Sequence,
		evt.TenantID, evt.UserID, evt.SessionID,
		evt.Context, evt.Payload, evt.PayloadHash, evt.EventHash,
	)

	return err
}
