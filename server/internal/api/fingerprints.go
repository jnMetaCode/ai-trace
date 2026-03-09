package api

import (
	"errors"
	"net/http"

	"github.com/ai-trace/server/internal/store"
	"github.com/gin-gonic/gin"
)

// GetFingerprint 获取指定 trace 的推理行为指纹
// @Summary 获取推理行为指纹
// @Description 获取指定 trace_id 的完整 4 层推理行为指纹
// @Tags Fingerprint
// @Produce json
// @Param trace_id path string true "追踪 ID"
// @Success 200 {object} FingerprintResponse "指纹数据"
// @Failure 404 {object} map[string]interface{} "未找到"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Security ApiKeyAuth
// @Router /fingerprints/{trace_id} [get]
func (h *Handler) GetFingerprint(c *gin.Context) {
	traceID := c.Param("trace_id")
	if traceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "trace_id is required",
		})
		return
	}

	tenantID := c.GetString("tenant_id")

	// 创建指纹存储
	fpStore := store.NewFingerprintStore(h.stores.DB)

	record, err := fpStore.GetByTraceID(c.Request.Context(), tenantID, traceID)
	if err != nil {
		if errors.Is(err, store.ErrFingerprintNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Fingerprint not found",
				"message": "No fingerprint found for this trace_id. Fingerprints are generated automatically during AI inference.",
				"suggestions": []string{
					"Verify the trace_id is correct",
					"Make an AI request first via POST /api/v1/chat/completions",
					"Fingerprints are only generated for traces with sufficient data",
				},
			})
			return
		}
		h.logger.Errorw("Failed to get fingerprint", "error", err, "trace_id", traceID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve fingerprint",
		})
		return
	}

	c.JSON(http.StatusOK, FingerprintResponse{
		TraceID:         record.TraceID,
		FingerprintID:   record.FingerprintID,
		FingerprintHash: record.FingerprintHash,
		ModelID:         record.ModelID,
		ModelProvider:   record.ModelProvider,
		Statistical:     record.StatisticalFeatures,
		TokenProbs:      record.TokenProbFeatures,
		Semantic:        record.SemanticFeatures,
		GeneratedAt:     record.GeneratedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// CompareFingerprints 比较两个 trace 的指纹相似度
// @Summary 比较指纹相似度
// @Description 比较两个 trace 的推理行为指纹相似度
// @Tags Fingerprint
// @Accept json
// @Produce json
// @Param request body FingerprintCompareRequest true "比较请求"
// @Success 200 {object} FingerprintCompareResponse "比较结果"
// @Failure 400 {object} map[string]interface{} "请求无效"
// @Failure 404 {object} map[string]interface{} "指纹未找到"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Security ApiKeyAuth
// @Router /fingerprints/compare [post]
func (h *Handler) CompareFingerprints(c *gin.Context) {
	var req FingerprintCompareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	tenantID := c.GetString("tenant_id")

	// 创建指纹存储
	fpStore := store.NewFingerprintStore(h.stores.DB)

	result, err := fpStore.Compare(c.Request.Context(), tenantID, req.TraceID1, req.TraceID2)
	if err != nil {
		if errors.Is(err, store.ErrFingerprintNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Fingerprints not found",
				"message": "One or both fingerprints not found. Both traces must have fingerprints to compare.",
				"suggestions": []string{
					"Verify both trace_id values are correct",
					"Use GET /api/v1/fingerprints/{trace_id} to check if each fingerprint exists",
					"Fingerprints are generated automatically during AI inference",
				},
			})
			return
		}
		h.logger.Errorw("Failed to compare fingerprints", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to compare fingerprints",
		})
		return
	}

	c.JSON(http.StatusOK, FingerprintCompareResponse{
		TraceID1:   result.TraceID1,
		TraceID2:   result.TraceID2,
		Similarity: result.OverallSimilarity,
		Details: FingerprintCompareDetails{
			StatisticalSimilarity: result.StatisticalSimilarity,
			SemanticSimilarity:    result.SemanticSimilarity,
			TokenProbSimilarity:   result.TokenProbSimilarity,
		},
		Conclusion: result.Conclusion,
	})
}

// VerifyFingerprint 验证指纹完整性
// @Summary 验证指纹完整性
// @Description 验证指定 trace 的指纹是否被篡改
// @Tags Fingerprint
// @Accept json
// @Produce json
// @Param request body FingerprintVerifyRequest true "验证请求"
// @Success 200 {object} FingerprintVerifyResponse "验证结果"
// @Failure 400 {object} map[string]interface{} "请求无效"
// @Failure 404 {object} map[string]interface{} "指纹未找到"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Security ApiKeyAuth
// @Router /fingerprints/verify [post]
func (h *Handler) VerifyFingerprint(c *gin.Context) {
	var req FingerprintVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	tenantID := c.GetString("tenant_id")

	// 创建指纹存储
	fpStore := store.NewFingerprintStore(h.stores.DB)

	result, err := fpStore.Verify(c.Request.Context(), tenantID, req.TraceID, req.FingerprintHash)
	if err != nil {
		if errors.Is(err, store.ErrFingerprintNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Fingerprint not found",
				"message": "No fingerprint found for this trace_id.",
				"suggestions": []string{
					"Verify the trace_id is correct",
					"Use GET /api/v1/fingerprints/{trace_id} to check if the fingerprint exists",
					"Fingerprints are generated automatically during AI inference",
				},
			})
			return
		}
		h.logger.Errorw("Failed to verify fingerprint", "error", err, "trace_id", req.TraceID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify fingerprint",
		})
		return
	}

	c.JSON(http.StatusOK, FingerprintVerifyResponse{
		TraceID:  result.TraceID,
		Valid:    result.Valid,
		Message:  result.Message,
		Verified: result.Valid,
	})
}

// FingerprintResponse 指纹响应
type FingerprintResponse struct {
	TraceID         string      `json:"trace_id"`
	FingerprintID   string      `json:"fingerprint_id"`
	FingerprintHash string      `json:"fingerprint_hash"`
	ModelID         string      `json:"model_id"`
	ModelProvider   string      `json:"model_provider,omitempty"`
	Statistical     interface{} `json:"statistical"`
	TokenProbs      interface{} `json:"token_probs,omitempty"`
	Semantic        interface{} `json:"semantic"`
	GeneratedAt     string      `json:"generated_at"`
}

// FingerprintCompareRequest 指纹比较请求
type FingerprintCompareRequest struct {
	TraceID1 string `json:"trace_id_1" binding:"required"`
	TraceID2 string `json:"trace_id_2" binding:"required"`
}

// FingerprintCompareResponse 指纹比较响应
type FingerprintCompareResponse struct {
	TraceID1   string                    `json:"trace_id_1"`
	TraceID2   string                    `json:"trace_id_2"`
	Similarity float64                   `json:"similarity"`
	Details    FingerprintCompareDetails `json:"details"`
	Conclusion string                    `json:"conclusion"`
}

// FingerprintCompareDetails 比较详情
type FingerprintCompareDetails struct {
	StatisticalSimilarity float64 `json:"statistical_similarity"`
	SemanticSimilarity    float64 `json:"semantic_similarity"`
	TokenProbSimilarity   float64 `json:"token_prob_similarity"`
}

// FingerprintVerifyRequest 指纹验证请求
type FingerprintVerifyRequest struct {
	TraceID         string `json:"trace_id" binding:"required"`
	FingerprintHash string `json:"fingerprint_hash,omitempty"`
}

// FingerprintVerifyResponse 指纹验证响应
type FingerprintVerifyResponse struct {
	TraceID  string `json:"trace_id"`
	Valid    bool   `json:"valid"`
	Message  string `json:"message"`
	Verified bool   `json:"verified"`
}
