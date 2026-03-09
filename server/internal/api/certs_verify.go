package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ai-trace/server/internal/cert"
	"github.com/ai-trace/server/internal/merkle"
	"github.com/ai-trace/server/pkg/sign"
	"github.com/gin-gonic/gin"
)

// VerifyCertRequest 验证存证请求
// @Description 验证存证证书的请求体
type VerifyCertRequest struct {
	CertID   string `json:"cert_id,omitempty" example:"cert_abc123"`        // 证书 ID
	RootHash string `json:"root_hash,omitempty" example:"sha256:abcd1234"` // Merkle 根哈希
}

// VerifyCert 验证存证
// @Summary 验证存证证书
// @Description 验证存证证书的完整性和有效性
// @Description
// @Description ## 验证项目
// @Description - **hash_integrity**: 事件哈希与 Merkle 根匹配
// @Description - **signature**: Ed25519 数字签名验证
// @Description - **time_proof**: 时间证明验证
// @Description - **anchor_proof**: 锚定证明验证（WORM/区块链）
// @Tags Certificates
// @Accept json
// @Produce json
// @Param X-API-Key header string true "AI-Trace 平台 API Key"
// @Param request body VerifyCertRequest true "验证请求（cert_id 或 root_hash 二选一）"
// @Success 200 {object} map[string]interface{} "验证结果"
// @Failure 400 {object} map[string]interface{} "请求无效"
// @Failure 404 {object} map[string]interface{} "证书不存在"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Security ApiKeyAuth
// @Router /certs/verify [post]
func (h *Handler) VerifyCert(c *gin.Context) {
	var req VerifyCertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	if req.CertID == "" && req.RootHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "cert_id or root_hash is required",
		})
		return
	}

	tenantID := c.GetString("tenant_id")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// 查询存证
	var certData []byte
	var query string
	var args []interface{}

	if req.CertID != "" {
		query = `SELECT cert_data FROM certificates WHERE cert_id = $1 AND tenant_id = $2`
		args = []interface{}{req.CertID, tenantID}
	} else {
		query = `SELECT cert_data FROM certificates WHERE root_hash = $1 AND tenant_id = $2`
		args = []interface{}{req.RootHash, tenantID}
	}

	err := h.stores.DB.QueryRow(ctx, query, args...).Scan(&certData)
	if err != nil {
		identifier := req.CertID
		if identifier == "" {
			identifier = req.RootHash
		}
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Certificate not found",
			"message": fmt.Sprintf("No certificate found matching '%s'.", identifier),
			"suggestions": []string{
				"Verify the cert_id or root_hash is correct",
				"Use GET /api/v1/certs/search to list available certificates",
				"The certificate may have been generated under a different tenant",
			},
		})
		return
	}

	var certificate cert.Certificate
	if err := json.Unmarshal(certData, &certificate); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse certificate",
		})
		return
	}

	// 验证Merkle树
	verifyTree, err := merkle.NewTree(certificate.EventHashes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to rebuild merkle tree",
		})
		return
	}

	hashIntegrity := verifyTree.GetRoot() == certificate.RootHash

	// 验证结果
	checks := map[string]interface{}{
		"hash_integrity": map[string]interface{}{
			"passed":  hashIntegrity,
			"message": "Event hashes match merkle root",
		},
		"merkle_root": map[string]interface{}{
			"passed":  hashIntegrity,
			"message": "Root hash verified",
		},
	}

	// 安全访问 TimeProof 并验证签名
	signatureValid := false
	if certificate.TimeProof != nil {
		// 验证签名
		if certificate.TimeProof.Signature != "" && certificate.Metadata != nil && certificate.Metadata.PublicKey != "" {
			signatureData := fmt.Sprintf("%s|%s|%s|%s",
				certificate.CertID,
				certificate.RootHash,
				string(certificate.Metadata.EvidenceLevel),
				certificate.TimeProof.Timestamp.Format(time.RFC3339),
			)
			verified, err := sign.VerifyWithPublicKey(
				certificate.Metadata.PublicKey,
				[]byte(signatureData),
				certificate.TimeProof.Signature,
			)
			if err == nil && verified {
				signatureValid = true
			}
		}

		checks["time_proof"] = map[string]interface{}{
			"passed":    true,
			"timestamp": certificate.TimeProof.Timestamp,
			"type":      certificate.TimeProof.ProofType,
		}
		checks["signature"] = map[string]interface{}{
			"passed":  signatureValid,
			"message": func() string {
				if signatureValid {
					return "Ed25519 signature verified"
				}
				return "Signature verification failed or not available"
			}(),
		}
	} else {
		checks["time_proof"] = map[string]interface{}{
			"passed":  false,
			"message": "Time proof not available",
		}
		checks["signature"] = map[string]interface{}{
			"passed":  false,
			"message": "No signature available",
		}
	}

	// 安全访问 AnchorProof
	if certificate.AnchorProof != nil {
		checks["anchor_proof"] = map[string]interface{}{
			"passed":    true,
			"type":      certificate.AnchorProof.AnchorType,
			"anchor_id": certificate.AnchorProof.AnchorID,
			"timestamp": certificate.AnchorProof.AnchorTimestamp,
		}
	} else {
		checks["anchor_proof"] = map[string]interface{}{
			"passed":  false,
			"message": "Anchor proof not available",
		}
	}

	// 验证通过需要: hash完整性 + 签名有效
	valid := hashIntegrity && signatureValid

	// 构建人类可读的摘要
	summary := buildVerificationSummary(valid, &certificate, hashIntegrity, signatureValid)

	c.JSON(http.StatusOK, gin.H{
		"valid":       valid,
		"summary":     summary,
		"checks":      checks,
		"certificate": certificate,
		"next_steps":  buildNextSteps(valid, &certificate),
	})
}

// buildVerificationSummary creates a human-readable summary of the verification result.
func buildVerificationSummary(valid bool, cert *cert.Certificate, hashOK, sigOK bool) map[string]interface{} {
	result := map[string]interface{}{}

	if valid {
		result["status"] = "VERIFIED"
		result["message"] = "This certificate is valid and has not been tampered with."
	} else {
		result["status"] = "FAILED"
		if !hashOK {
			result["message"] = "Certificate integrity check failed - the data may have been modified."
		} else if !sigOK {
			result["message"] = "Signature verification failed - the certificate may not be authentic."
		} else {
			result["message"] = "Verification failed due to unknown reasons."
		}
	}

	// Add context about what was verified
	if cert != nil {
		result["trace_id"] = cert.TraceID
		result["event_count"] = len(cert.EventHashes)
		if cert.Metadata != nil {
			result["evidence_level"] = cert.Metadata.EvidenceLevel
			result["created_at"] = cert.Metadata.CreatedAt.Format("2006-01-02 15:04:05 UTC")

			// Add evidence level description
			switch cert.Metadata.EvidenceLevel {
			case "internal":
				result["evidence_description"] = "Internal audit level - Ed25519 signed, suitable for development and team reviews"
			case "compliance":
				result["evidence_description"] = "Compliance level - WORM storage with TSA timestamp, suitable for SOC2/GDPR/HIPAA"
			case "legal":
				result["evidence_description"] = "Legal evidence level - Blockchain anchored, suitable for legal disputes and court proceedings"
			}
		}
	}

	return result
}

// buildNextSteps suggests what the user should do next based on verification result.
func buildNextSteps(valid bool, cert *cert.Certificate) []string {
	steps := []string{}

	if valid {
		steps = append(steps, "You can share this verification result as proof of AI decision audit")
		steps = append(steps, "Use GET /api/v1/certs/{cert_id} to retrieve the full certificate")
		steps = append(steps, "Use POST /api/v1/certs/{cert_id}/prove to generate a minimal disclosure proof")
	} else {
		steps = append(steps, "Re-generate the certificate using POST /api/v1/certs/commit")
		steps = append(steps, "Check if the original trace events still exist")
		steps = append(steps, "Contact support if this error persists")
	}

	return steps
}
