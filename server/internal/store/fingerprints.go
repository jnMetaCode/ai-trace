package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ai-trace/server/internal/fingerprint"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	// ErrFingerprintNotFound 指纹未找到
	ErrFingerprintNotFound = errors.New("fingerprint not found")
)

// FingerprintStore 指纹存储
type FingerprintStore struct {
	db *pgxpool.Pool
}

// FingerprintRecord 指纹数据库记录
type FingerprintRecord struct {
	ID                    int64                             `json:"id"`
	FingerprintID         string                            `json:"fingerprint_id"`
	TraceID               string                            `json:"trace_id"`
	TenantID              string                            `json:"tenant_id"`
	CertID                *string                           `json:"cert_id,omitempty"`
	FingerprintHash       string                            `json:"fingerprint_hash"`
	ModelID               string                            `json:"model_id"`
	ModelProvider         string                            `json:"model_provider"`
	StatisticalFeatures   *fingerprint.StatisticalFeatures  `json:"statistical_features"`
	TokenProbFeatures     *fingerprint.TokenProbFeatures    `json:"token_prob_features,omitempty"`
	ModelInternalFeatures *fingerprint.ModelInternalFeatures `json:"model_internal_features,omitempty"`
	SemanticFeatures      *fingerprint.SemanticFeatures     `json:"semantic_features"`
	FullFingerprint       *fingerprint.InferenceFingerprint `json:"full_fingerprint"`
	Status                string                            `json:"status"`
	GeneratedAt           time.Time                         `json:"generated_at"`
	CreatedAt             time.Time                         `json:"created_at"`
}

// NewFingerprintStore 创建指纹存储
func NewFingerprintStore(db *pgxpool.Pool) *FingerprintStore {
	return &FingerprintStore{db: db}
}

// Save 保存指纹
func (s *FingerprintStore) Save(ctx context.Context, tenantID, traceID string, fp *fingerprint.InferenceFingerprint) (*FingerprintRecord, error) {
	fingerprintID := fmt.Sprintf("fp_%s", uuid.New().String()[:8])

	// 序列化 JSON 字段
	statisticalJSON, err := json.Marshal(fp.Statistical)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal statistical features: %w", err)
	}

	semanticJSON, err := json.Marshal(fp.Semantic)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal semantic features: %w", err)
	}

	fullJSON, err := json.Marshal(fp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal full fingerprint: %w", err)
	}

	// 可选字段
	var tokenProbJSON, modelInternalJSON []byte
	if fp.TokenProbs != nil {
		tokenProbJSON, _ = json.Marshal(fp.TokenProbs)
	}
	if fp.ModelInternal != nil {
		modelInternalJSON, _ = json.Marshal(fp.ModelInternal)
	}

	query := `
		INSERT INTO fingerprints (
			fingerprint_id, trace_id, tenant_id, fingerprint_hash,
			model_id, model_provider,
			statistical_features, token_prob_features, model_internal_features, semantic_features,
			full_fingerprint, status, generated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at
	`

	var id int64
	var createdAt time.Time

	err = s.db.QueryRow(ctx, query,
		fingerprintID, traceID, tenantID, fp.FingerprintHash,
		fp.ModelID, fp.ModelProvider,
		statisticalJSON, tokenProbJSON, modelInternalJSON, semanticJSON,
		fullJSON, "active", fp.GeneratedAt,
	).Scan(&id, &createdAt)

	if err != nil {
		return nil, fmt.Errorf("failed to insert fingerprint: %w", err)
	}

	return &FingerprintRecord{
		ID:                  id,
		FingerprintID:       fingerprintID,
		TraceID:             traceID,
		TenantID:            tenantID,
		FingerprintHash:     fp.FingerprintHash,
		ModelID:             fp.ModelID,
		ModelProvider:       fp.ModelProvider,
		StatisticalFeatures: fp.Statistical,
		TokenProbFeatures:   fp.TokenProbs,
		SemanticFeatures:    fp.Semantic,
		FullFingerprint:     fp,
		Status:              "active",
		GeneratedAt:         fp.GeneratedAt,
		CreatedAt:           createdAt,
	}, nil
}

// GetByTraceID 通过 trace_id 获取指纹
func (s *FingerprintStore) GetByTraceID(ctx context.Context, tenantID, traceID string) (*FingerprintRecord, error) {
	query := `
		SELECT id, fingerprint_id, trace_id, tenant_id, cert_id, fingerprint_hash,
			   model_id, model_provider,
			   statistical_features, token_prob_features, model_internal_features, semantic_features,
			   full_fingerprint, status, generated_at, created_at
		FROM fingerprints
		WHERE trace_id = $1 AND tenant_id = $2 AND status = 'active'
		ORDER BY created_at DESC
		LIMIT 1
	`

	return s.scanRecord(ctx, query, traceID, tenantID)
}

// GetByFingerprintID 通过 fingerprint_id 获取指纹
func (s *FingerprintStore) GetByFingerprintID(ctx context.Context, fingerprintID string) (*FingerprintRecord, error) {
	query := `
		SELECT id, fingerprint_id, trace_id, tenant_id, cert_id, fingerprint_hash,
			   model_id, model_provider,
			   statistical_features, token_prob_features, model_internal_features, semantic_features,
			   full_fingerprint, status, generated_at, created_at
		FROM fingerprints
		WHERE fingerprint_id = $1 AND status = 'active'
	`

	return s.scanRecord(ctx, query, fingerprintID)
}

// scanRecord 扫描单条记录
func (s *FingerprintStore) scanRecord(ctx context.Context, query string, args ...interface{}) (*FingerprintRecord, error) {
	row := s.db.QueryRow(ctx, query, args...)

	var record FingerprintRecord
	var certID, modelProvider *string
	var statisticalJSON, tokenProbJSON, modelInternalJSON, semanticJSON, fullJSON []byte

	err := row.Scan(
		&record.ID, &record.FingerprintID, &record.TraceID, &record.TenantID, &certID,
		&record.FingerprintHash, &record.ModelID, &modelProvider,
		&statisticalJSON, &tokenProbJSON, &modelInternalJSON, &semanticJSON,
		&fullJSON, &record.Status, &record.GeneratedAt, &record.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrFingerprintNotFound
		}
		return nil, fmt.Errorf("failed to scan fingerprint: %w", err)
	}

	record.CertID = certID
	if modelProvider != nil {
		record.ModelProvider = *modelProvider
	}

	// 反序列化 JSON 字段
	if statisticalJSON != nil {
		record.StatisticalFeatures = &fingerprint.StatisticalFeatures{}
		json.Unmarshal(statisticalJSON, record.StatisticalFeatures)
	}
	if tokenProbJSON != nil {
		record.TokenProbFeatures = &fingerprint.TokenProbFeatures{}
		json.Unmarshal(tokenProbJSON, record.TokenProbFeatures)
	}
	if modelInternalJSON != nil {
		record.ModelInternalFeatures = &fingerprint.ModelInternalFeatures{}
		json.Unmarshal(modelInternalJSON, record.ModelInternalFeatures)
	}
	if semanticJSON != nil {
		record.SemanticFeatures = &fingerprint.SemanticFeatures{}
		json.Unmarshal(semanticJSON, record.SemanticFeatures)
	}
	if fullJSON != nil {
		record.FullFingerprint = &fingerprint.InferenceFingerprint{}
		json.Unmarshal(fullJSON, record.FullFingerprint)
	}

	return &record, nil
}

// Compare 比较两个指纹的相似度
func (s *FingerprintStore) Compare(ctx context.Context, tenantID, traceID1, traceID2 string) (*FingerprintCompareResult, error) {
	fp1, err := s.GetByTraceID(ctx, tenantID, traceID1)
	if err != nil {
		return nil, fmt.Errorf("failed to get fingerprint 1: %w", err)
	}

	fp2, err := s.GetByTraceID(ctx, tenantID, traceID2)
	if err != nil {
		return nil, fmt.Errorf("failed to get fingerprint 2: %w", err)
	}

	// 使用 fingerprint 包的比较函数
	similarity := fingerprint.CompareFingerprints(fp1.FullFingerprint, fp2.FullFingerprint)

	// 计算各层相似度
	var statSim, tokenSim, semanticSim float64

	if fp1.StatisticalFeatures != nil && fp2.StatisticalFeatures != nil {
		statSim = compareStatistical(fp1.StatisticalFeatures, fp2.StatisticalFeatures)
	}
	if fp1.TokenProbFeatures != nil && fp2.TokenProbFeatures != nil {
		tokenSim = compareTokenProb(fp1.TokenProbFeatures, fp2.TokenProbFeatures)
	}
	if fp1.SemanticFeatures != nil && fp2.SemanticFeatures != nil {
		semanticSim = compareSemantic(fp1.SemanticFeatures, fp2.SemanticFeatures)
	}

	return &FingerprintCompareResult{
		TraceID1:              traceID1,
		TraceID2:              traceID2,
		OverallSimilarity:     similarity,
		StatisticalSimilarity: statSim,
		TokenProbSimilarity:   tokenSim,
		SemanticSimilarity:    semanticSim,
		Conclusion:            getSimilarityConclusion(similarity),
	}, nil
}

// FingerprintCompareResult 指纹比较结果
type FingerprintCompareResult struct {
	TraceID1              string  `json:"trace_id_1"`
	TraceID2              string  `json:"trace_id_2"`
	OverallSimilarity     float64 `json:"overall_similarity"`
	StatisticalSimilarity float64 `json:"statistical_similarity"`
	TokenProbSimilarity   float64 `json:"token_prob_similarity"`
	SemanticSimilarity    float64 `json:"semantic_similarity"`
	Conclusion            string  `json:"conclusion"`
}

// Verify 验证指纹完整性
func (s *FingerprintStore) Verify(ctx context.Context, tenantID, traceID string, expectedHash string) (*FingerprintVerifyResult, error) {
	record, err := s.GetByTraceID(ctx, tenantID, traceID)
	if err != nil {
		return nil, err
	}

	// 检查 FullFingerprint 是否存在
	if record.FullFingerprint == nil {
		return &FingerprintVerifyResult{
			TraceID:      traceID,
			Valid:        false,
			StoredHash:   record.FingerprintHash,
			ComputedHash: "",
			ExpectedHash: expectedHash,
			Message:      "Fingerprint data is missing or corrupted",
		}, nil
	}

	// 重新计算哈希
	computedHash := record.FullFingerprint.ComputeFingerprintHash()

	// 验证逻辑：
	// 1. 存储的哈希必须与计算的哈希一致（验证数据完整性）
	// 2. 如果提供了 expectedHash，还需验证与存储的哈希一致
	hashIntegrity := record.FingerprintHash == computedHash
	hashMatch := expectedHash == "" || record.FingerprintHash == expectedHash
	isValid := hashIntegrity && hashMatch

	result := &FingerprintVerifyResult{
		TraceID:      traceID,
		Valid:        isValid,
		StoredHash:   record.FingerprintHash,
		ComputedHash: computedHash,
		ExpectedHash: expectedHash,
	}

	if !isValid {
		if !hashIntegrity {
			result.Message = "Fingerprint data has been tampered with"
		} else if !hashMatch {
			result.Message = "Expected hash does not match stored hash"
		}
	} else {
		result.Message = "Fingerprint integrity verified"
	}

	return result, nil
}

// FingerprintVerifyResult 指纹验证结果
type FingerprintVerifyResult struct {
	TraceID      string `json:"trace_id"`
	Valid        bool   `json:"valid"`
	StoredHash   string `json:"stored_hash"`
	ComputedHash string `json:"computed_hash"`
	ExpectedHash string `json:"expected_hash"`
	Message      string `json:"message"`
}

// UpdateCertID 更新指纹关联的证书 ID
func (s *FingerprintStore) UpdateCertID(ctx context.Context, traceID, certID string) error {
	query := `UPDATE fingerprints SET cert_id = $1 WHERE trace_id = $2 AND status = 'active'`
	_, err := s.db.Exec(ctx, query, certID, traceID)
	return err
}

// List 列出指纹
func (s *FingerprintStore) List(ctx context.Context, tenantID string, limit, offset int) ([]*FingerprintRecord, int64, error) {
	// 获取总数
	var total int64
	countQuery := `SELECT COUNT(*) FROM fingerprints WHERE tenant_id = $1 AND status = 'active'`
	err := s.db.QueryRow(ctx, countQuery, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count fingerprints: %w", err)
	}

	// 获取列表
	query := `
		SELECT id, fingerprint_id, trace_id, tenant_id, cert_id, fingerprint_hash,
			   model_id, model_provider,
			   statistical_features, token_prob_features, model_internal_features, semantic_features,
			   full_fingerprint, status, generated_at, created_at
		FROM fingerprints
		WHERE tenant_id = $1 AND status = 'active'
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query fingerprints: %w", err)
	}
	defer rows.Close()

	var records []*FingerprintRecord
	for rows.Next() {
		var record FingerprintRecord
		var certID, modelProvider *string
		var statisticalJSON, tokenProbJSON, modelInternalJSON, semanticJSON, fullJSON []byte

		err := rows.Scan(
			&record.ID, &record.FingerprintID, &record.TraceID, &record.TenantID, &certID,
			&record.FingerprintHash, &record.ModelID, &modelProvider,
			&statisticalJSON, &tokenProbJSON, &modelInternalJSON, &semanticJSON,
			&fullJSON, &record.Status, &record.GeneratedAt, &record.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan row: %w", err)
		}

		record.CertID = certID
		if modelProvider != nil {
			record.ModelProvider = *modelProvider
		}

		// 反序列化 JSON
		if statisticalJSON != nil {
			record.StatisticalFeatures = &fingerprint.StatisticalFeatures{}
			json.Unmarshal(statisticalJSON, record.StatisticalFeatures)
		}
		if fullJSON != nil {
			record.FullFingerprint = &fingerprint.InferenceFingerprint{}
			json.Unmarshal(fullJSON, record.FullFingerprint)
		}

		records = append(records, &record)
	}

	return records, total, nil
}

// 辅助函数：比较统计特征
func compareStatistical(a, b *fingerprint.StatisticalFeatures) float64 {
	if a == nil || b == nil {
		return 0
	}

	// 计算各指标的相似度
	tpsSim := 1.0 - minFloat(absFloat(a.TokensPerSecond-b.TokensPerSecond)/maxFloat(a.TokensPerSecond, b.TokensPerSecond), 1.0)
	ftSim := 1.0 - minFloat(absFloat(float64(a.FirstTokenMs-b.FirstTokenMs))/maxFloat(float64(a.FirstTokenMs), float64(b.FirstTokenMs)), 1.0)

	return (tpsSim + ftSim) / 2
}

// 辅助函数：比较 token 概率特征
func compareTokenProb(a, b *fingerprint.TokenProbFeatures) float64 {
	if a == nil || b == nil {
		return 0
	}

	avgSim := 1.0 - minFloat(absFloat(a.AvgLogProb-b.AvgLogProb)/maxFloat(absFloat(a.AvgLogProb), absFloat(b.AvgLogProb)), 1.0)
	entropySim := 1.0 - minFloat(absFloat(a.AvgEntropy-b.AvgEntropy)/maxFloat(a.AvgEntropy, b.AvgEntropy), 1.0)

	return (avgSim + entropySim) / 2
}

// 辅助函数：比较语义特征
func compareSemantic(a, b *fingerprint.SemanticFeatures) float64 {
	if a == nil || b == nil {
		return 0
	}

	vocabSim := 1.0 - absFloat(a.VocabularyDiversity-b.VocabularyDiversity)
	sentSim := 1.0 - minFloat(absFloat(a.AvgSentenceLength-b.AvgSentenceLength)/maxFloat(a.AvgSentenceLength, b.AvgSentenceLength), 1.0)
	complexSim := 1.0 - minFloat(absFloat(a.TextComplexity-b.TextComplexity)/100, 1.0)

	return (vocabSim + sentSim + complexSim) / 3
}

// 辅助函数
func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func getSimilarityConclusion(similarity float64) string {
	switch {
	case similarity >= 0.95:
		return "highly_similar"
	case similarity >= 0.8:
		return "similar"
	case similarity >= 0.6:
		return "moderately_similar"
	case similarity >= 0.4:
		return "somewhat_different"
	default:
		return "very_different"
	}
}

// ComputeFingerprintHash 计算指纹哈希（包装函数用于测试）
func ComputeFingerprintHash(fp *fingerprint.InferenceFingerprint) string {
	if fp == nil {
		return ""
	}
	return fp.ComputeFingerprintHash()
}
