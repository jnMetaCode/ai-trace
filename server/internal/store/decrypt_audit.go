package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DecryptAuditStore 解密审计日志存储
type DecryptAuditStore struct {
	db *pgxpool.Pool
}

// DecryptAuditRecord 解密审计记录
type DecryptAuditRecord struct {
	ID           int64     `json:"id"`
	AuditLogID   string    `json:"audit_id"`
	ContentID    string    `json:"content_id,omitempty"`
	EncryptedRef string    `json:"encrypted_ref"`
	ContentType  string    `json:"content_type"`
	TenantID     string    `json:"tenant_id"`
	UserID       string    `json:"user_id"`
	ClientIP     string    `json:"client_ip"`
	UserAgent    string    `json:"user_agent,omitempty"`
	RequestID    string    `json:"request_id,omitempty"`
	TraceID      string    `json:"trace_id,omitempty"` // 可选，用于关联
	Success      bool      `json:"success"`
	FailReason   string    `json:"error_message,omitempty"`
	DecryptedAt  time.Time `json:"decrypted_at"`
}

// NewDecryptAuditStore 创建解密审计存储
func NewDecryptAuditStore(db *pgxpool.Pool) *DecryptAuditStore {
	return &DecryptAuditStore{db: db}
}

// Log 记录解密审计日志
func (s *DecryptAuditStore) Log(ctx context.Context, record *DecryptAuditRecord) error {
	if record.AuditLogID == "" {
		record.AuditLogID = uuid.New().String()
	}
	if record.ContentID == "" {
		// 使用 encrypted_ref 生成一个 content_id
		record.ContentID = fmt.Sprintf("cnt_%s", uuid.New().String()[:8])
	}
	if record.DecryptedAt.IsZero() {
		record.DecryptedAt = time.Now()
	}

	query := `
		INSERT INTO decrypt_audit_logs (
			audit_id, content_id, encrypted_ref, content_type, tenant_id, user_id,
			client_ip, user_agent, request_id, success, error_message, decrypted_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	// request_id 可以用 trace_id 填充
	requestID := record.RequestID
	if requestID == "" {
		requestID = record.TraceID
	}

	err := s.db.QueryRow(ctx, query,
		record.AuditLogID,
		record.ContentID,
		record.EncryptedRef,
		record.ContentType,
		record.TenantID,
		record.UserID,
		record.ClientIP,
		record.UserAgent,
		requestID,
		record.Success,
		record.FailReason,
		record.DecryptedAt,
	).Scan(&record.ID)

	if err != nil {
		return fmt.Errorf("failed to insert decrypt audit log: %w", err)
	}

	return nil
}

// GetByEncryptedRef 获取指定加密引用的审计日志
func (s *DecryptAuditStore) GetByEncryptedRef(ctx context.Context, tenantID, encryptedRef string, limit, offset int) ([]*DecryptAuditRecord, int64, error) {
	// 获取总数
	var total int64
	countQuery := `SELECT COUNT(*) FROM decrypt_audit_logs WHERE tenant_id = $1 AND encrypted_ref = $2`
	err := s.db.QueryRow(ctx, countQuery, tenantID, encryptedRef).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// 获取列表
	query := `
		SELECT id, audit_id, content_id, encrypted_ref, content_type, tenant_id, user_id,
			   client_ip, user_agent, request_id, success, error_message, decrypted_at
		FROM decrypt_audit_logs
		WHERE tenant_id = $1 AND encrypted_ref = $2
		ORDER BY decrypted_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := s.db.Query(ctx, query, tenantID, encryptedRef, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var records []*DecryptAuditRecord
	for rows.Next() {
		var r DecryptAuditRecord
		var userID, clientIP, userAgent, requestID, failReason *string

		err := rows.Scan(
			&r.ID, &r.AuditLogID, &r.ContentID, &r.EncryptedRef, &r.ContentType, &r.TenantID, &userID,
			&clientIP, &userAgent, &requestID, &r.Success, &failReason, &r.DecryptedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan row: %w", err)
		}

		if userID != nil {
			r.UserID = *userID
		}
		if clientIP != nil {
			r.ClientIP = *clientIP
		}
		if userAgent != nil {
			r.UserAgent = *userAgent
		}
		if requestID != nil {
			r.RequestID = *requestID
			r.TraceID = *requestID
		}
		if failReason != nil {
			r.FailReason = *failReason
		}

		records = append(records, &r)
	}

	return records, total, nil
}

// GetByUserID 获取用户的解密审计日志
func (s *DecryptAuditStore) GetByUserID(ctx context.Context, tenantID, userID string, limit, offset int) ([]*DecryptAuditRecord, int64, error) {
	// 获取总数
	var total int64
	countQuery := `SELECT COUNT(*) FROM decrypt_audit_logs WHERE tenant_id = $1 AND user_id = $2`
	err := s.db.QueryRow(ctx, countQuery, tenantID, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// 获取列表
	query := `
		SELECT id, audit_id, content_id, encrypted_ref, content_type, tenant_id, user_id,
			   client_ip, user_agent, request_id, success, error_message, decrypted_at
		FROM decrypt_audit_logs
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY decrypted_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := s.db.Query(ctx, query, tenantID, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var records []*DecryptAuditRecord
	for rows.Next() {
		var r DecryptAuditRecord
		var uid, clientIP, userAgent, requestID, failReason *string

		err := rows.Scan(
			&r.ID, &r.AuditLogID, &r.ContentID, &r.EncryptedRef, &r.ContentType, &r.TenantID, &uid,
			&clientIP, &userAgent, &requestID, &r.Success, &failReason, &r.DecryptedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan row: %w", err)
		}

		if uid != nil {
			r.UserID = *uid
		}
		if clientIP != nil {
			r.ClientIP = *clientIP
		}
		if userAgent != nil {
			r.UserAgent = *userAgent
		}
		if requestID != nil {
			r.RequestID = *requestID
			r.TraceID = *requestID
		}
		if failReason != nil {
			r.FailReason = *failReason
		}

		records = append(records, &r)
	}

	return records, total, nil
}

// GetByContentID 获取 content_id 的解密审计日志
func (s *DecryptAuditStore) GetByContentID(ctx context.Context, tenantID, contentID string) ([]*DecryptAuditRecord, error) {
	query := `
		SELECT id, audit_id, content_id, encrypted_ref, content_type, tenant_id, user_id,
			   client_ip, user_agent, request_id, success, error_message, decrypted_at
		FROM decrypt_audit_logs
		WHERE tenant_id = $1 AND content_id = $2
		ORDER BY decrypted_at DESC
	`

	rows, err := s.db.Query(ctx, query, tenantID, contentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var records []*DecryptAuditRecord
	for rows.Next() {
		var r DecryptAuditRecord
		var uid, clientIP, userAgent, requestID, failReason *string

		err := rows.Scan(
			&r.ID, &r.AuditLogID, &r.ContentID, &r.EncryptedRef, &r.ContentType, &r.TenantID, &uid,
			&clientIP, &userAgent, &requestID, &r.Success, &failReason, &r.DecryptedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if uid != nil {
			r.UserID = *uid
		}
		if clientIP != nil {
			r.ClientIP = *clientIP
		}
		if userAgent != nil {
			r.UserAgent = *userAgent
		}
		if requestID != nil {
			r.RequestID = *requestID
			r.TraceID = *requestID
		}
		if failReason != nil {
			r.FailReason = *failReason
		}

		records = append(records, &r)
	}

	return records, nil
}
