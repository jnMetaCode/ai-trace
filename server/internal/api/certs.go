package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ai-trace/server/internal/cert"
	"github.com/ai-trace/server/internal/merkle"
	"github.com/ai-trace/server/internal/store"
	"github.com/ai-trace/server/pkg/sign"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CommitCertRequest 生成存证请求
// @Description 生成存证证书的请求体
type CommitCertRequest struct {
	TraceID       string `json:"trace_id" binding:"required" example:"trc_abc123"` // 追踪 ID
	EvidenceLevel string `json:"evidence_level,omitempty" example:"internal"`      // 存证级别：internal/compliance/legal（或旧格式 L1/L2/L3）
}

// CommitCert 生成存证
// @Summary 生成存证证书
// @Description 为指定的追踪生成存证证书，支持多级存证
// @Description
// @Description ## 存证级别
// @Description - **internal** (L1): Ed25519 签名，适合内部审计
// @Description - **compliance** (L2): WORM 存储 + TSA 时间戳，适合合规要求
// @Description - **legal** (L3): 区块链锚定，适合法律效力要求
// @Tags Certificates
// @Accept json
// @Produce json
// @Param X-API-Key header string true "AI-Trace 平台 API Key"
// @Param request body CommitCertRequest true "存证请求"
// @Success 200 {object} map[string]interface{} "存证证书信息"
// @Failure 400 {object} map[string]interface{} "请求无效"
// @Failure 404 {object} map[string]interface{} "追踪不存在"
// @Failure 401 {object} map[string]interface{} "未授权"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Security ApiKeyAuth
// @Router /certs/commit [post]
func (h *Handler) CommitCert(c *gin.Context) {
	var req CommitCertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	tenantID := c.GetString("tenant_id")

	// 获取trace下的所有事件
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	rows, err := h.stores.DB.Query(ctx, `
		SELECT event_hash FROM events
		WHERE trace_id = $1 AND tenant_id = $2
		ORDER BY sequence ASC
	`, req.TraceID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to query events",
		})
		return
	}
	defer rows.Close()

	eventHashes := make([]string, 0)
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			h.logger.Warnf("Failed to scan event hash: %v", err)
			continue
		}
		eventHashes = append(eventHashes, hash)
	}

	// 检查行迭代错误
	if err := rows.Err(); err != nil {
		h.logger.Errorf("Error iterating event rows: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read events",
		})
		return
	}

	if len(eventHashes) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "No events found for trace_id",
			"message": fmt.Sprintf("No events were found for trace_id '%s'. This could mean the trace doesn't exist or hasn't been recorded yet.", req.TraceID),
			"suggestions": []string{
				"Verify the trace_id is correct (should start with 'trc_')",
				"Check if the AI request was made successfully",
				"Use GET /api/v1/events/search?trace_id=xxx to verify events exist",
			},
		})
		return
	}

	// 构建Merkle树
	tree, err := merkle.NewTree(eventHashes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to build merkle tree",
		})
		return
	}

	// 确定存证级别（支持新旧命名格式）
	evidenceLevel := cert.ParseEvidenceLevel(req.EvidenceLevel)
	if evidenceLevel == "" {
		evidenceLevel = cert.EvidenceLevelInternal
	}

	// 获取签名器
	signer, err := sign.GetDefaultSigner()
	if err != nil {
		h.logger.Errorf("Failed to get signer: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Signing service unavailable",
		})
		return
	}

	// 生成证书ID和时间戳
	certID := fmt.Sprintf("cert_%s", uuid.New().String()[:12])
	createdAt := time.Now()

	// 签名证书（签名内容: cert_id + root_hash + evidence_level + timestamp）
	signature, err := signer.SignCertificate(
		certID,
		tree.GetRoot(),
		string(evidenceLevel),
		createdAt.Format(time.RFC3339),
	)
	if err != nil {
		h.logger.Errorf("Failed to sign certificate: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to sign certificate",
		})
		return
	}

	// 生成存证
	certificate := &cert.Certificate{
		CertID:        certID,
		CertVersion:   "1.0",
		SchemaVersion: "0.1",
		TraceID:       req.TraceID,
		EventHashes:   eventHashes,
		MerkleTree:    tree,
		RootHash:      tree.GetRoot(),
		TimeProof: &cert.TimeProof{
			ProofType: "local",
			Timestamp: createdAt,
			Signature: signature,
		},
		AnchorProof: &cert.AnchorProof{
			AnchorType:      "local",
			AnchorID:        fmt.Sprintf("anchor_%s", uuid.New().String()[:8]),
			AnchorTimestamp: createdAt,
		},
		Metadata: &cert.CertMetadata{
			TenantID:      tenantID,
			CreatedAt:     createdAt,
			CreatedBy:     "ai-trace-server/v0.1",
			EvidenceLevel: evidenceLevel,
			PublicKey:     signer.GetPublicKeyHex(),
		},
	}

	// Compliance级别：WORM存储
	if evidenceLevel == cert.EvidenceLevelCompliance {
		certificate.AnchorProof.AnchorType = "worm"
		certificate.AnchorProof.StorageProvider = "minio"

		// 写入MinIO并设置对象锁（WORM）
		if h.stores.Minio != nil {
			wormStorage := store.NewWORMStorage(h.stores.Minio, h.config.Minio.Bucket, 365)
			wormResult, wormErr := wormStorage.StoreCertificate(ctx, certificate.CertID, certificate)
			if wormErr != nil {
				h.logger.Warnf("Failed to store certificate in WORM storage: %v", wormErr)
			} else {
				certificate.AnchorProof.ObjectKey = wormResult.ObjectKey
				certificate.AnchorProof.AnchorID = wormResult.ETag
				h.logger.Infof("Certificate stored in WORM: %s (retention until %s)",
					wormResult.ObjectKey, wormResult.RetentionDate.Format(time.RFC3339))
			}
		} else {
			h.logger.Warn("MinIO not configured, skipping WORM storage for L2 certificate")
		}
	}

	// 存储存证
	certJSON, err := json.Marshal(certificate)
	if err != nil {
		h.logger.Errorf("Failed to marshal certificate: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to serialize certificate",
		})
		return
	}
	_, err = h.stores.DB.Exec(ctx, `
		INSERT INTO certificates (
			cert_id, trace_id, tenant_id, root_hash,
			event_count, evidence_level, cert_data, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		certificate.CertID, certificate.TraceID, tenantID,
		certificate.RootHash, len(eventHashes), evidenceLevel,
		certJSON, time.Now(),
	)
	if err != nil {
		h.logger.Errorf("Failed to store certificate: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to store certificate",
		})
		return
	}

	// 构建人类可读的存证级别描述
	levelDescription := ""
	switch evidenceLevel {
	case cert.EvidenceLevelInternal:
		levelDescription = "Internal audit level - Ed25519 signed, instant generation"
	case cert.EvidenceLevelCompliance:
		levelDescription = "Compliance level - WORM storage with TSA timestamp"
	case cert.EvidenceLevelLegal:
		levelDescription = "Legal evidence level - Blockchain anchored"
	}

	c.JSON(http.StatusOK, gin.H{
		"cert_id":        certificate.CertID,
		"trace_id":       certificate.TraceID,
		"root_hash":      certificate.RootHash,
		"event_count":    len(eventHashes),
		"evidence_level": evidenceLevel,
		"evidence_description": levelDescription,
		"time_proof":     certificate.TimeProof,
		"anchor_proof":   certificate.AnchorProof,
		"created_at":     certificate.Metadata.CreatedAt,
		"next_steps": []string{
			fmt.Sprintf("Verify: curl -X POST /api/v1/certs/verify -d '{\"cert_id\":\"%s\"}'", certificate.CertID),
			fmt.Sprintf("Retrieve: curl /api/v1/certs/%s", certificate.CertID),
			fmt.Sprintf("Generate proof: curl -X POST /api/v1/certs/%s/prove", certificate.CertID),
		},
	})
}
