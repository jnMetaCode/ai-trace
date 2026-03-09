package api

import (
	"net/http"
	"os"

	"github.com/ai-trace/server/internal/anchor"
	"github.com/ai-trace/server/internal/cert"
	"github.com/ai-trace/server/internal/config"
	"github.com/ai-trace/server/internal/crypto"
	"github.com/ai-trace/server/internal/gateway"
	"github.com/ai-trace/server/internal/metrics"
	"github.com/ai-trace/server/internal/middleware"
	"github.com/ai-trace/server/internal/report"
	"github.com/ai-trace/server/internal/store"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

// Handler API处理器
type Handler struct {
	config          *config.Config
	stores          *store.Stores
	gateway         *gateway.Gateway
	logger          *zap.SugaredLogger
	reportGen       *report.Generator
	federatedAnchor *anchor.FederatedAnchor
	encryptedStore  *store.EncryptedStore
	keystore        *crypto.MemoryKeyStore
	autoCertEval    *cert.AutoCertEvaluator
}

// NewRouter 创建路由
func NewRouter(cfg *config.Config, stores *store.Stores, gw *gateway.Gateway, logger *zap.SugaredLogger) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// 安全头
	r.Use(middleware.SecurityHeaders())

	// CORS 中间件
	r.Use(middleware.CORS())

	// 请求 ID
	r.Use(middleware.RequestID())

	// AI-Trace 信息头
	r.Use(middleware.TraceHeadersMiddleware())

	// 速率限制
	rateLimiter := middleware.NewRateLimiter(middleware.RateLimiterConfig{
		RequestsPerMinute: 120, // 每分钟 120 次请求
		BurstSize:         20,  // 突发 20 次
	})
	r.Use(middleware.RateLimit(rateLimiter))

	// 添加 Prometheus 监控中间件（如果启用）
	if cfg.Features.Metrics {
		r.Use(metrics.Middleware())
		metrics.RegisterEndpoint(r)
		logger.Info("Prometheus metrics enabled at /metrics")
	}

	// 初始化自动存证评估器
	var autoCertStrategy *cert.AutoCertStrategy
	if cfg.AutoCert.Enabled {
		autoCertStrategy = &cert.AutoCertStrategy{
			Enabled:      true,
			DefaultLevel: cert.ParseEvidenceLevel(cfg.AutoCert.DefaultLevel),
			Triggers: []cert.AutoCertTrigger{
				{
					Type:        "model",
					Description: "Auto-cert for configured models",
					Condition: cert.AutoCertCondition{
						Models: cfg.AutoCert.Models,
					},
					Level: cert.ParseEvidenceLevel(cfg.AutoCert.DefaultLevel),
				},
				{
					Type:        "token_count",
					Description: "Auto-cert for large responses",
					Condition: cert.AutoCertCondition{
						MinTokens: cfg.AutoCert.MinTokens,
					},
					Level: cert.EvidenceLevelInternal,
				},
			},
		}
	}
	autoCertEval := cert.NewAutoCertEvaluator(autoCertStrategy, logger)

	handler := &Handler{
		config:       cfg,
		stores:       stores,
		gateway:      gw,
		logger:       logger,
		autoCertEval: autoCertEval,
	}

	// 初始化密钥存储和加密存储
	kekStr := os.Getenv("AI_TRACE_KEK")
	if kekStr == "" {
		// 检查是否为生产环境
		ginMode := os.Getenv("GIN_MODE")
		if ginMode == "release" {
			logger.Error("CRITICAL: AI_TRACE_KEK environment variable must be set in production mode")
			// 在生产环境中不使用默认 KEK，这将导致加密功能不可用
		} else {
			// 仅在开发/测试环境使用默认 KEK
			kekStr = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
			logger.Warn("Using default KEK - set AI_TRACE_KEK env for production")
		}
	}
	keystore, err := crypto.NewMemoryKeyStoreFromString(kekStr)
	if err != nil {
		logger.Warnf("Failed to initialize keystore: %v", err)
	} else {
		handler.keystore = keystore
		handler.encryptedStore = store.NewEncryptedStore(stores.Minio, cfg.Minio.Bucket, keystore)
	}

	// 初始化报告生成器（如果启用）
	if cfg.Features.Reports {
		reportGen, err := report.NewGenerator()
		if err != nil {
			logger.Warnf("Failed to initialize report generator: %v", err)
		} else {
			handler.reportGen = reportGen
		}
	}

	// 初始化联邦锚定器（如果启用）
	if cfg.Features.FederatedNodes && len(cfg.Anchor.Federated.Nodes) > 0 {
		anchorCfg := &anchor.Config{
			FederatedNodes:   cfg.Anchor.Federated.Nodes,
			MinConfirmations: cfg.Anchor.Federated.MinConfirmations,
		}
		fedAnchor, err := anchor.NewFederatedAnchor(anchorCfg, logger)
		if err != nil {
			logger.Warnf("Failed to initialize federated anchor: %v", err)
		} else {
			handler.federatedAnchor = fedAnchor
			logger.Infof("Federated anchor initialized, node ID: %s", fedAnchor.GetNodeID())
		}
	}

	// 健康检查端点
	r.GET("/health", handler.Health)
	r.GET("/health/detailed", handler.HealthDetailed)
	r.GET("/ready", handler.Ready)
	r.GET("/live", handler.Live)

	// Swagger API 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1
	v1 := r.Group("/api/v1")
	{
		// 认证中间件
		v1.Use(handler.AuthMiddleware())

		// 快速入门指南
		v1.GET("/getting-started", handler.GettingStarted)

		// Gateway代理 - OpenAI兼容
		v1.POST("/chat/completions", handler.ChatCompletions)
		// 流式聊天（SSE）- 增量存证
		v1.POST("/chat/completions/stream", handler.ChatCompletionsStream)

		// 事件API
		events := v1.Group("/events")
		{
			events.POST("/ingest", handler.IngestEvents)
			events.GET("/search", handler.SearchEvents)
			events.GET("/:event_id", handler.GetEvent)
		}

		// 存证API
		certs := v1.Group("/certs")
		{
			certs.POST("/commit", handler.CommitCert)
			certs.POST("/verify", handler.VerifyCert)
			certs.GET("/search", handler.SearchCerts)
			certs.GET("/:cert_id", handler.GetCert)
			certs.POST("/:cert_id/prove", handler.GenerateProof)
		}

		// 报告API
		reports := v1.Group("/reports")
		{
			reports.POST("/generate", handler.GenerateReport)
		}

		// 指纹API
		fingerprints := v1.Group("/fingerprints")
		{
			fingerprints.GET("/:trace_id", handler.GetFingerprint)
			fingerprints.POST("/compare", handler.CompareFingerprints)
			fingerprints.POST("/verify", handler.VerifyFingerprint)
		}

		// 解密API
		decrypt := v1.Group("/decrypt")
		{
			decrypt.POST("", handler.DecryptContent)
			decrypt.GET("/audit", handler.GetDecryptAuditLogs)
		}

		// 联邦节点 API（需要联邦功能启用）
		if handler.federatedAnchor != nil {
			handler.RegisterFederatedRoutes(v1)
		}
	}

	return r
}

// AuthMiddleware 认证中间件
func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			apiKey = c.GetHeader("Authorization")
			if len(apiKey) > 7 && apiKey[:7] == "Bearer " {
				apiKey = apiKey[7:]
			}
		}

		// 验证API Key
		valid := false
		for _, key := range h.config.Auth.APIKeys {
			if key == apiKey {
				valid = true
				break
			}
		}

		if !valid && len(h.config.Auth.APIKeys) > 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid API key",
			})
			c.Abort()
			return
		}

		// 从header获取租户信息
		tenantID := c.GetHeader("X-Tenant-ID")
		if tenantID == "" {
			tenantID = "default"
		}
		c.Set("tenant_id", tenantID)

		userID := c.GetHeader("X-User-ID")
		c.Set("user_id", userID)

		c.Next()
	}
}

