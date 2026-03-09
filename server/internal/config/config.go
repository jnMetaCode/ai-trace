package config

import (
	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	// DeployMode: "standard" (PostgreSQL+Redis+MinIO) or "simple" (SQLite only)
	DeployMode string         `mapstructure:"deploy_mode"`
	Server     ServerConfig   `mapstructure:"server"`
	Database   DatabaseConfig `mapstructure:"database"`
	Redis      RedisConfig    `mapstructure:"redis"`
	Minio      MinioConfig    `mapstructure:"minio"`
	SQLite     SQLiteConfig   `mapstructure:"sqlite"`
	Gateway    GatewayConfig  `mapstructure:"gateway"`
	Auth       AuthConfig     `mapstructure:"auth"`
	Anchor     AnchorConfig   `mapstructure:"anchor"`
	Features   FeatureConfig  `mapstructure:"features"`
	AutoCert   AutoCertConfig `mapstructure:"auto_cert"`
}

// SQLiteConfig SQLite配置（simple mode）
type SQLiteConfig struct {
	Path string `mapstructure:"path"`
}

// AutoCertConfig 自动证书配置
type AutoCertConfig struct {
	Enabled      bool     `mapstructure:"enabled"`
	DefaultLevel string   `mapstructure:"default_level"`
	Models       []string `mapstructure:"models"`
	MinTokens    int      `mapstructure:"min_tokens"`
	Schedule     string   `mapstructure:"schedule"`
}

// FeatureConfig 功能开关配置
type FeatureConfig struct {
	// L3 区块链锚定（需要 -tags blockchain 编译）
	BlockchainAnchor bool `mapstructure:"blockchain_anchor"`
	// 联邦化验证节点
	FederatedNodes bool `mapstructure:"federated_nodes"`
	// Prometheus 监控
	Metrics bool `mapstructure:"metrics"`
	// 报告生成
	Reports bool `mapstructure:"reports"`
}

// AnchorConfig 锚定配置
type AnchorConfig struct {
	// 以太坊配置
	Ethereum EthereumConfig `mapstructure:"ethereum"`
	// Polygon配置
	Polygon PolygonConfig `mapstructure:"polygon"`
	// 联邦节点配置
	Federated FederatedConfig `mapstructure:"federated"`
}

// EthereumConfig 以太坊配置
type EthereumConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	RPCURL          string `mapstructure:"rpc_url"`
	PrivateKey      string `mapstructure:"private_key"`
	ChainID         int64  `mapstructure:"chain_id"`
	ContractAddress string `mapstructure:"contract_address"`
	GasLimit        uint64 `mapstructure:"gas_limit"`
	MaxGasPrice     uint64 `mapstructure:"max_gas_price"`
}

// PolygonConfig Polygon配置
type PolygonConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	RPCURL          string `mapstructure:"rpc_url"`
	PrivateKey      string `mapstructure:"private_key"`
	ChainID         int64  `mapstructure:"chain_id"`
	ContractAddress string `mapstructure:"contract_address"`
}

// FederatedConfig 联邦节点配置
type FederatedConfig struct {
	Enabled          bool     `mapstructure:"enabled"`
	Nodes            []string `mapstructure:"nodes"`
	MinConfirmations int      `mapstructure:"min_confirmations"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // debug, release
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// MinioConfig MinIO配置
type MinioConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Bucket    string `mapstructure:"bucket"`
	UseSSL    bool   `mapstructure:"use_ssl"`
}

// GatewayConfig Gateway配置
type GatewayConfig struct {
	// OpenAI配置
	OpenAI OpenAIConfig `mapstructure:"openai"`
	// Ollama配置
	Ollama OllamaConfig `mapstructure:"ollama"`
	// 默认超时（秒）
	Timeout int `mapstructure:"timeout"`
	// 最大重试次数
	MaxRetries int `mapstructure:"max_retries"`
}

// OpenAIConfig OpenAI API配置
type OpenAIConfig struct {
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"api_key"`
}

// OllamaConfig Ollama配置
type OllamaConfig struct {
	BaseURL string `mapstructure:"base_url"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	APIKeys []string `mapstructure:"api_keys"`
}

// ValidationError 配置验证错误
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// ValidationErrors 多个验证错误
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	msg := "config validation failed:\n"
	for _, err := range e {
		msg += "  - " + err.Error() + "\n"
	}
	return msg
}

// IsSimpleMode returns true if running in simple (SQLite) mode
func (c *Config) IsSimpleMode() bool {
	return c.DeployMode == "simple"
}

// Validate 验证配置
func (c *Config) Validate() error {
	var errs ValidationErrors

	// 验证部署模式
	if c.DeployMode != "" && c.DeployMode != "standard" && c.DeployMode != "simple" {
		errs = append(errs, ValidationError{
			Field:   "deploy_mode",
			Message: "must be 'standard' or 'simple'",
		})
	}

	// 验证服务器配置
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errs = append(errs, ValidationError{
			Field:   "server.port",
			Message: "must be between 1 and 65535",
		})
	}
	if c.Server.Mode != "debug" && c.Server.Mode != "release" && c.Server.Mode != "test" {
		errs = append(errs, ValidationError{
			Field:   "server.mode",
			Message: "must be 'debug', 'release', or 'test'",
		})
	}

	// Simple mode only requires SQLite path
	if c.IsSimpleMode() {
		if c.SQLite.Path == "" {
			errs = append(errs, ValidationError{
				Field:   "sqlite.path",
				Message: "is required in simple mode",
			})
		}
		// Skip PostgreSQL/Redis/MinIO validation in simple mode
		if len(errs) > 0 {
			return errs
		}
		return nil
	}

	// Standard mode requires PostgreSQL, Redis, MinIO
	// 验证数据库配置
	if c.Database.Host == "" {
		errs = append(errs, ValidationError{
			Field:   "database.host",
			Message: "is required",
		})
	}
	if c.Database.Port < 1 || c.Database.Port > 65535 {
		errs = append(errs, ValidationError{
			Field:   "database.port",
			Message: "must be between 1 and 65535",
		})
	}
	if c.Database.DBName == "" {
		errs = append(errs, ValidationError{
			Field:   "database.dbname",
			Message: "is required",
		})
	}

	// 验证 Redis 配置
	if c.Redis.Host == "" {
		errs = append(errs, ValidationError{
			Field:   "redis.host",
			Message: "is required",
		})
	}
	if c.Redis.Port < 1 || c.Redis.Port > 65535 {
		errs = append(errs, ValidationError{
			Field:   "redis.port",
			Message: "must be between 1 and 65535",
		})
	}

	// 验证 MinIO 配置
	if c.Minio.Endpoint == "" {
		errs = append(errs, ValidationError{
			Field:   "minio.endpoint",
			Message: "is required",
		})
	}
	if c.Minio.Bucket == "" {
		errs = append(errs, ValidationError{
			Field:   "minio.bucket",
			Message: "is required",
		})
	}

	// 验证 Gateway 配置
	if c.Gateway.Timeout < 1 {
		errs = append(errs, ValidationError{
			Field:   "gateway.timeout",
			Message: "must be at least 1 second",
		})
	}
	if c.Gateway.MaxRetries < 0 {
		errs = append(errs, ValidationError{
			Field:   "gateway.max_retries",
			Message: "cannot be negative",
		})
	}

	// 验证区块链配置（如果启用）
	if c.Features.BlockchainAnchor {
		if c.Anchor.Ethereum.Enabled && c.Anchor.Ethereum.RPCURL == "" {
			errs = append(errs, ValidationError{
				Field:   "anchor.ethereum.rpc_url",
				Message: "is required when ethereum anchor is enabled",
			})
		}
		if c.Anchor.Polygon.Enabled && c.Anchor.Polygon.RPCURL == "" {
			errs = append(errs, ValidationError{
				Field:   "anchor.polygon.rpc_url",
				Message: "is required when polygon anchor is enabled",
			})
		}
	}

	// 验证联邦节点配置（如果启用）
	if c.Features.FederatedNodes {
		if c.Anchor.Federated.MinConfirmations < 1 {
			errs = append(errs, ValidationError{
				Field:   "anchor.federated.min_confirmations",
				Message: "must be at least 1",
			})
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// Load 加载配置
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/ai-trace")

	// 设置默认值
	setDefaults()

	// 环境变量覆盖
	viper.AutomaticEnv()

	// 绑定特定环境变量（支持 docker-compose 配置）
	viper.BindEnv("deploy_mode", "AI_TRACE_MODE")
	viper.BindEnv("sqlite.path", "AI_TRACE_DB_PATH")
	viper.BindEnv("server.port", "AI_TRACE_PORT")
	viper.BindEnv("auth.api_keys", "AI_TRACE_DEFAULT_API_KEY")

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		// 配置文件不存在时使用默认值
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func setDefaults() {
	// Deploy mode (standard by default)
	viper.SetDefault("deploy_mode", "standard")

	// Server
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")

	// SQLite (for simple mode)
	viper.SetDefault("sqlite.path", "./data/ai-trace.db")

	// Database
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.dbname", "ai_trace")
	viper.SetDefault("database.sslmode", "disable")

	// Redis
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// MinIO
	viper.SetDefault("minio.endpoint", "localhost:9000")
	viper.SetDefault("minio.access_key", "minioadmin")
	viper.SetDefault("minio.secret_key", "minioadmin")
	viper.SetDefault("minio.bucket", "ai-trace")
	viper.SetDefault("minio.use_ssl", false)

	// Gateway
	viper.SetDefault("gateway.openai.base_url", "https://api.openai.com/v1")
	viper.SetDefault("gateway.ollama.base_url", "http://localhost:11434")
	viper.SetDefault("gateway.timeout", 120)
	viper.SetDefault("gateway.max_retries", 3)

	// Features (默认关闭高级功能)
	viper.SetDefault("features.blockchain_anchor", false)
	viper.SetDefault("features.federated_nodes", false)
	viper.SetDefault("features.metrics", false)
	viper.SetDefault("features.reports", true)

	// Anchor - Ethereum (默认关闭)
	viper.SetDefault("anchor.ethereum.enabled", false)
	viper.SetDefault("anchor.ethereum.chain_id", 1)
	viper.SetDefault("anchor.ethereum.gas_limit", 100000)
	viper.SetDefault("anchor.ethereum.max_gas_price", 100000000000) // 100 Gwei

	// Anchor - Polygon (默认关闭)
	viper.SetDefault("anchor.polygon.enabled", false)
	viper.SetDefault("anchor.polygon.chain_id", 137)

	// Anchor - Federated (默认关闭)
	viper.SetDefault("anchor.federated.enabled", false)
	viper.SetDefault("anchor.federated.min_confirmations", 2)

	// Auto-cert (默认关闭)
	viper.SetDefault("auto_cert.enabled", false)
	viper.SetDefault("auto_cert.default_level", "internal")
	viper.SetDefault("auto_cert.models", []string{"gpt-4", "claude-3-opus"})
	viper.SetDefault("auto_cert.min_tokens", 2000)
	viper.SetDefault("auto_cert.schedule", "daily")
}
