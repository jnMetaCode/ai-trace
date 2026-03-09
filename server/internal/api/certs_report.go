package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ai-trace/server/internal/report"
	"github.com/gin-gonic/gin"
)

// GenerateReportRequest 报告生成请求
// @Description 生成审计报告的请求体
type GenerateReportRequest struct {
	Type       string   `json:"type" binding:"required" example:"audit"`      // 报告类型: audit/compliance/summary
	Format     string   `json:"format,omitempty" example:"html"`              // 输出格式: json/html
	TraceIDs   []string `json:"trace_ids,omitempty"`                          // 指定追踪 ID 列表
	CertIDs    []string `json:"cert_ids,omitempty"`                           // 指定证书 ID 列表
	StartTime  string   `json:"start_time,omitempty" example:"2024-01-01"`    // 开始时间
	EndTime    string   `json:"end_time,omitempty" example:"2024-12-31"`      // 结束时间
	IncludeRaw bool     `json:"include_raw,omitempty"`                        // 是否包含原始数据
}

// GenerateReport 生成报告
// @Summary 生成审计报告
// @Description 生成 AI 决策审计报告，支持多种类型和格式
// @Description
// @Description ## 报告类型
// @Description - **audit**: 完整审计报告，包含所有追踪和证书详情
// @Description - **compliance**: 合规性报告，重点关注证书验证状态
// @Description - **summary**: 摘要报告，统计汇总信息
// @Description
// @Description ## 输出格式
// @Description - **json**: JSON 格式，适合程序处理
// @Description - **html**: HTML 格式，适合直接查看和打印
// @Tags Reports
// @Accept json
// @Produce json,html
// @Param X-API-Key header string true "AI-Trace 平台 API Key"
// @Param request body GenerateReportRequest true "报告请求"
// @Success 200 {object} report.Report "报告信息"
// @Failure 400 {object} map[string]interface{} "请求无效"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Security ApiKeyAuth
// @Router /reports/generate [post]
func (h *Handler) GenerateReport(c *gin.Context) {
	// 检查报告功能是否启用
	if h.reportGen == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Report generation is not enabled",
		})
		return
	}

	var req GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	tenantID := c.GetString("tenant_id")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	// 解析时间范围
	var startTime, endTime *time.Time
	if req.StartTime != "" {
		if t, err := time.Parse("2006-01-02", req.StartTime); err == nil {
			startTime = &t
		}
	}
	if req.EndTime != "" {
		if t, err := time.Parse("2006-01-02", req.EndTime); err == nil {
			endTime = &t
		}
	}

	// 确定报告类型和格式
	reportType := report.ReportTypeSummary
	switch req.Type {
	case "audit":
		reportType = report.ReportTypeAudit
	case "compliance":
		reportType = report.ReportTypeCompliance
	}

	reportFormat := report.ReportFormatJSON
	if req.Format == "html" {
		reportFormat = report.ReportFormatHTML
	}

	// 构建报告请求
	reportReq := &report.ReportRequest{
		Type:       reportType,
		Format:     reportFormat,
		TraceIDs:   req.TraceIDs,
		CertIDs:    req.CertIDs,
		StartTime:  startTime,
		EndTime:    endTime,
		TenantID:   tenantID,
		IncludeRaw: req.IncludeRaw,
	}

	// 收集报告数据
	reportData, err := h.collectReportData(ctx, reportReq)
	if err != nil {
		h.logger.Errorf("Failed to collect report data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to collect report data",
		})
		return
	}

	// 生成报告
	generatedReport, err := h.reportGen.Generate(ctx, reportReq, reportData)
	if err != nil {
		h.logger.Errorf("Failed to generate report: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to generate report: %v", err),
		})
		return
	}

	// 根据格式返回内容
	if reportFormat == report.ReportFormatHTML {
		c.Data(http.StatusOK, "text/html; charset=utf-8", generatedReport.Content)
	} else {
		c.JSON(http.StatusOK, generatedReport)
	}
}

// collectReportData 收集报告数据
func (h *Handler) collectReportData(ctx context.Context, req *report.ReportRequest) (*report.ReportData, error) {
	data := &report.ReportData{
		Summary: &report.Summary{
			EventsByType: make(map[string]int),
			CertsByLevel: make(map[string]int),
		},
		Details: &report.Details{
			Traces:        make([]report.TraceDetail, 0),
			Certificates:  make([]report.CertDetail, 0),
			Verifications: make([]report.Verification, 0),
		},
	}

	// 构建查询条件
	whereClause := "tenant_id = $1"
	args := []interface{}{req.TenantID}
	argIndex := 2

	if len(req.TraceIDs) > 0 {
		whereClause += fmt.Sprintf(" AND trace_id = ANY($%d)", argIndex)
		args = append(args, req.TraceIDs)
		argIndex++
	}
	if req.StartTime != nil {
		whereClause += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, req.StartTime)
		argIndex++
	}
	if req.EndTime != nil {
		whereClause += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, req.EndTime)
		argIndex++
	}

	// 查询追踪统计
	traceQuery := fmt.Sprintf(`
		SELECT trace_id, COUNT(*) as event_count,
		       MIN(created_at) as start_time, MAX(created_at) as end_time
		FROM events
		WHERE %s
		GROUP BY trace_id
		ORDER BY MIN(created_at) DESC
		LIMIT 1000
	`, whereClause)

	rows, err := h.stores.DB.Query(ctx, traceQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query traces: %w", err)
	}
	defer rows.Close()

	var minTime, maxTime time.Time
	totalEvents := 0

	for rows.Next() {
		var traceID string
		var eventCount int
		var startTime, endTime time.Time

		if err := rows.Scan(&traceID, &eventCount, &startTime, &endTime); err != nil {
			continue
		}

		totalEvents += eventCount
		data.Summary.TotalTraces++

		// 更新时间范围
		if minTime.IsZero() || startTime.Before(minTime) {
			minTime = startTime
		}
		if endTime.After(maxTime) {
			maxTime = endTime
		}

		if req.IncludeRaw {
			data.Details.Traces = append(data.Details.Traces, report.TraceDetail{
				TraceID:    traceID,
				EventCount: eventCount,
				StartTime:  startTime,
				EndTime:    endTime,
				Duration:   endTime.Sub(startTime).String(),
			})
		}
	}

	data.Summary.TotalEvents = totalEvents
	if !minTime.IsZero() {
		data.Summary.TimeRange = &report.TimeRange{
			Start: minTime,
			End:   maxTime,
		}
	}

	// 查询证书
	certWhereClause := "tenant_id = $1"
	certArgs := []interface{}{req.TenantID}
	certArgIndex := 2

	if len(req.CertIDs) > 0 {
		certWhereClause += fmt.Sprintf(" AND cert_id = ANY($%d)", certArgIndex)
		certArgs = append(certArgs, req.CertIDs)
		certArgIndex++
	}
	if len(req.TraceIDs) > 0 {
		certWhereClause += fmt.Sprintf(" AND trace_id = ANY($%d)", certArgIndex)
		certArgs = append(certArgs, req.TraceIDs)
		certArgIndex++
	}

	certQuery := fmt.Sprintf(`
		SELECT cert_id, trace_id, root_hash, evidence_level, created_at
		FROM certificates
		WHERE %s
		ORDER BY created_at DESC
		LIMIT 500
	`, certWhereClause)

	certRows, err := h.stores.DB.Query(ctx, certQuery, certArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query certificates: %w", err)
	}
	defer certRows.Close()

	for certRows.Next() {
		var certID, traceID, rootHash, evidenceLevel string
		var createdAt time.Time

		if err := certRows.Scan(&certID, &traceID, &rootHash, &evidenceLevel, &createdAt); err != nil {
			continue
		}

		data.Summary.TotalCertificates++
		data.Summary.CertsByLevel[evidenceLevel]++

		if req.IncludeRaw {
			data.Details.Certificates = append(data.Details.Certificates, report.CertDetail{
				CertID:        certID,
				TraceID:       traceID,
				RootHash:      rootHash,
				EvidenceLevel: evidenceLevel,
				CreatedAt:     createdAt,
				Verified:      true,
			})
		}
	}

	// 查询事件类型统计
	eventTypeQuery := fmt.Sprintf(`
		SELECT event_type, COUNT(*) as count
		FROM events
		WHERE %s
		GROUP BY event_type
	`, whereClause)

	typeRows, err := h.stores.DB.Query(ctx, eventTypeQuery, args[:len(args)]...)
	if err == nil {
		defer typeRows.Close()
		for typeRows.Next() {
			var eventType string
			var count int
			if err := typeRows.Scan(&eventType, &count); err == nil {
				data.Summary.EventsByType[eventType] = count
			}
		}
	}

	// 查询 Token 使用量
	tokenQuery := fmt.Sprintf(`
		SELECT COALESCE(SUM((metadata->>'prompt_tokens')::int), 0) as prompt_tokens,
		       COALESCE(SUM((metadata->>'completion_tokens')::int), 0) as completion_tokens
		FROM events
		WHERE %s AND event_type = 'llm_response'
	`, whereClause)

	var promptTokens, completionTokens int64
	if err := h.stores.DB.QueryRow(ctx, tokenQuery, args...).Scan(&promptTokens, &completionTokens); err == nil {
		data.Summary.TokenUsage = &report.TokenUsage{
			TotalPromptTokens:     promptTokens,
			TotalCompletionTokens: completionTokens,
			TotalTokens:           promptTokens + completionTokens,
		}
	}

	return data, nil
}
