package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ai-trace/server/internal/cert"
	"github.com/gin-gonic/gin"
)

// GenerateProofRequest 生成证明请求
// @Description 生成最小披露证明的请求体
type GenerateProofRequest struct {
	DiscloseEvents []int    `json:"disclose_events"` // 要披露的事件索引（如 [0, 2]）
	DiscloseFields []string `json:"disclose_fields"` // 要披露的字段（如 ["output_hash"]）
}

// GenerateProof 生成最小披露证明
// @Summary 生成最小披露证明
// @Description 生成只包含必要信息的证明，支持选择性披露
// @Description
// @Description ## 使用场景
// @Description - 向第三方证明 AI 决策过程，而不暴露完整对话内容
// @Description - 选择性披露特定事件和字段
// @Description - 生成可独立验证的 Merkle 证明
// @Tags Certificates
// @Accept json
// @Produce json
// @Param X-API-Key header string true "AI-Trace 平台 API Key"
// @Param cert_id path string true "证书 ID"
// @Param request body GenerateProofRequest true "证明请求"
// @Success 200 {object} cert.MinimalDisclosureProof "最小披露证明"
// @Failure 400 {object} map[string]interface{} "请求无效"
// @Failure 404 {object} map[string]interface{} "证书不存在"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Security ApiKeyAuth
// @Router /certs/{cert_id}/prove [post]
func (h *Handler) GenerateProof(c *gin.Context) {
	certID := c.Param("cert_id")
	tenantID := c.GetString("tenant_id")

	var req GenerateProofRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// 获取存证
	var certData []byte
	err := h.stores.DB.QueryRow(ctx, `
		SELECT cert_data FROM certificates WHERE cert_id = $1 AND tenant_id = $2
	`, certID, tenantID).Scan(&certData)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Certificate not found",
			"message": "No certificate found with the specified ID.",
			"suggestions": []string{
				"Verify the cert_id is correct (should start with 'cert_')",
				"Use GET /api/v1/certs/search to list available certificates",
				"Generate a certificate first using POST /api/v1/certs/commit",
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

	// 生成Merkle证明
	merkleProofs := make([]cert.EventMerkleProof, 0)
	for _, idx := range req.DiscloseEvents {
		if idx >= 0 && idx < len(certificate.EventHashes) {
			proof, err := certificate.MerkleTree.GetProof(idx)
			if err != nil {
				continue
			}
			merkleProofs = append(merkleProofs, cert.EventMerkleProof{
				EventIndex: idx,
				EventHash:  certificate.EventHashes[idx],
				ProofPath:  proof.Path,
			})
		}
	}

	// 构建最小披露证明
	minimalProof := &cert.MinimalDisclosureProof{
		SchemaVersion:   "0.1",
		CertID:          certificate.CertID,
		RootHash:        certificate.RootHash,
		DisclosedEvents: make([]cert.DisclosedEvent, 0),
		MerkleProofs:    merkleProofs,
		TimeProof:       certificate.TimeProof,
		AnchorProof:     certificate.AnchorProof,
		VerificationInstructions: &cert.VerificationInstructions{
			VerifierURL:   "https://github.com/ai-trace/verifier",
			VerifyCommand: "ai-trace verify --proof proof.json",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"proof": minimalProof,
		"summary": map[string]interface{}{
			"cert_id":         certificate.CertID,
			"disclosed_count": len(merkleProofs),
			"total_events":    len(certificate.EventHashes),
			"disclosure_rate": fmt.Sprintf("%.1f%%", float64(len(merkleProofs))/float64(len(certificate.EventHashes))*100),
		},
		"next_steps": []string{
			"Share this proof with third parties for verification",
			"The proof can be verified independently without the full certificate",
			"Use 'ai-trace verify --proof proof.json' to verify offline",
		},
	})
}
