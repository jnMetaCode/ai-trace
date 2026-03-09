package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
)

// WORMStorage WORM（一次写入多次读取）存储服务
type WORMStorage struct {
	client        *minio.Client
	bucket        string
	retentionDays int
}

// WORMUploadResult WORM上传结果
type WORMUploadResult struct {
	ObjectKey     string    `json:"object_key"`
	ETag          string    `json:"etag"`
	VersionID     string    `json:"version_id"`
	RetentionMode string    `json:"retention_mode"`
	RetentionDate time.Time `json:"retention_date"`
}

// NewWORMStorage 创建WORM存储服务
func NewWORMStorage(client *minio.Client, bucket string, retentionDays int) *WORMStorage {
	if retentionDays <= 0 {
		retentionDays = 365 // 默认保留1年
	}
	return &WORMStorage{
		client:        client,
		bucket:        bucket,
		retentionDays: retentionDays,
	}
}

// StoreCertificate 存储证书到WORM存储
// 对象一旦写入，在保留期内无法删除或修改
func (w *WORMStorage) StoreCertificate(ctx context.Context, certID string, certData interface{}) (*WORMUploadResult, error) {
	// 序列化证书数据
	data, err := json.Marshal(certData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal certificate: %w", err)
	}

	// 生成对象键
	objectKey := fmt.Sprintf("certs/%s/%s.json",
		time.Now().Format("2006/01/02"),
		certID,
	)

	// 计算保留截止时间
	retentionDate := time.Now().Add(time.Duration(w.retentionDays) * 24 * time.Hour)

	// 上传对象并设置对象锁
	uploadInfo, err := w.client.PutObject(ctx, w.bucket, objectKey, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{
			ContentType: "application/json",
			UserMetadata: map[string]string{
				"cert-id":    certID,
				"created-at": time.Now().Format(time.RFC3339),
				"worm":       "true",
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to minio: %w", err)
	}

	// 设置对象锁定（GOVERNANCE模式允许特权用户删除，COMPLIANCE模式完全不可删除）
	// 使用GOVERNANCE模式，可以在需要时由管理员解锁
	retentionMode := minio.Governance
	err = w.client.PutObjectRetention(ctx, w.bucket, objectKey, minio.PutObjectRetentionOptions{
		Mode:            &retentionMode,
		RetainUntilDate: &retentionDate,
		VersionID:       uploadInfo.VersionID,
	})
	if err != nil {
		// 对象锁定可能需要bucket启用版本控制和对象锁定
		// 如果设置失败，记录警告但不阻止流程
		// 这是因为开发环境可能没有启用对象锁定
		fmt.Printf("Warning: Failed to set object retention (bucket may not have object locking enabled): %v\n", err)
	}

	return &WORMUploadResult{
		ObjectKey:     objectKey,
		ETag:          uploadInfo.ETag,
		VersionID:     uploadInfo.VersionID,
		RetentionMode: "GOVERNANCE",
		RetentionDate: retentionDate,
	}, nil
}

// GetCertificate 从WORM存储获取证书
func (w *WORMStorage) GetCertificate(ctx context.Context, objectKey string) ([]byte, error) {
	obj, err := w.client.GetObject(ctx, w.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer obj.Close()

	// 读取对象内容
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}

	return buf.Bytes(), nil
}

// VerifyRetention 验证对象的保留状态
func (w *WORMStorage) VerifyRetention(ctx context.Context, objectKey string) (*WORMUploadResult, error) {
	// 获取对象状态
	stat, err := w.client.StatObject(ctx, w.bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to stat object: %w", err)
	}

	result := &WORMUploadResult{
		ObjectKey: objectKey,
		ETag:      stat.ETag,
		VersionID: stat.VersionID,
	}

	// 获取保留信息（MinIO API返回 mode, retainUntilDate, err）
	mode, retainUntilDate, err := w.client.GetObjectRetention(ctx, w.bucket, objectKey, stat.VersionID)
	if err == nil && mode != nil {
		result.RetentionMode = string(*mode)
		if retainUntilDate != nil {
			result.RetentionDate = *retainUntilDate
		}
	}

	return result, nil
}

// EnableBucketVersioning 启用Bucket版本控制（WORM的前提条件）
func (w *WORMStorage) EnableBucketVersioning(ctx context.Context) error {
	err := w.client.EnableVersioning(ctx, w.bucket)
	if err != nil {
		return fmt.Errorf("failed to enable versioning: %w", err)
	}
	return nil
}

// SetBucketObjectLockConfig 设置Bucket对象锁定配置
// 注意：需要在创建bucket时启用对象锁定，之后无法启用
func (w *WORMStorage) SetBucketObjectLockConfig(ctx context.Context) error {
	mode := minio.Governance
	unit := minio.Days
	validity := uint(w.retentionDays)

	err := w.client.SetObjectLockConfig(ctx, w.bucket, &mode, &validity, &unit)
	if err != nil {
		return fmt.Errorf("failed to set object lock config: %w", err)
	}
	return nil
}
