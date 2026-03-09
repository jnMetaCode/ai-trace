package config

import (
	"testing"
)

// TestValidateServerConfig 测试服务器配置验证
func TestValidateServerConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: Config{
				Server:   ServerConfig{Port: 8080, Mode: "debug"},
				Database: DatabaseConfig{Host: "localhost", Port: 5432, DBName: "test"},
				Redis:    RedisConfig{Host: "localhost", Port: 6379},
				Minio:    MinioConfig{Endpoint: "localhost:9000", Bucket: "test"},
				Gateway:  GatewayConfig{Timeout: 30, MaxRetries: 3},
			},
			wantErr: false,
		},
		{
			name: "invalid port - zero",
			cfg: Config{
				Server:   ServerConfig{Port: 0, Mode: "debug"},
				Database: DatabaseConfig{Host: "localhost", Port: 5432, DBName: "test"},
				Redis:    RedisConfig{Host: "localhost", Port: 6379},
				Minio:    MinioConfig{Endpoint: "localhost:9000", Bucket: "test"},
				Gateway:  GatewayConfig{Timeout: 30, MaxRetries: 3},
			},
			wantErr: true,
			errMsg:  "server.port",
		},
		{
			name: "invalid port - too high",
			cfg: Config{
				Server:   ServerConfig{Port: 70000, Mode: "debug"},
				Database: DatabaseConfig{Host: "localhost", Port: 5432, DBName: "test"},
				Redis:    RedisConfig{Host: "localhost", Port: 6379},
				Minio:    MinioConfig{Endpoint: "localhost:9000", Bucket: "test"},
				Gateway:  GatewayConfig{Timeout: 30, MaxRetries: 3},
			},
			wantErr: true,
			errMsg:  "server.port",
		},
		{
			name: "invalid mode",
			cfg: Config{
				Server:   ServerConfig{Port: 8080, Mode: "invalid"},
				Database: DatabaseConfig{Host: "localhost", Port: 5432, DBName: "test"},
				Redis:    RedisConfig{Host: "localhost", Port: 6379},
				Minio:    MinioConfig{Endpoint: "localhost:9000", Bucket: "test"},
				Gateway:  GatewayConfig{Timeout: 30, MaxRetries: 3},
			},
			wantErr: true,
			errMsg:  "server.mode",
		},
		{
			name: "missing database host",
			cfg: Config{
				Server:   ServerConfig{Port: 8080, Mode: "debug"},
				Database: DatabaseConfig{Host: "", Port: 5432, DBName: "test"},
				Redis:    RedisConfig{Host: "localhost", Port: 6379},
				Minio:    MinioConfig{Endpoint: "localhost:9000", Bucket: "test"},
				Gateway:  GatewayConfig{Timeout: 30, MaxRetries: 3},
			},
			wantErr: true,
			errMsg:  "database.host",
		},
		{
			name: "missing database name",
			cfg: Config{
				Server:   ServerConfig{Port: 8080, Mode: "debug"},
				Database: DatabaseConfig{Host: "localhost", Port: 5432, DBName: ""},
				Redis:    RedisConfig{Host: "localhost", Port: 6379},
				Minio:    MinioConfig{Endpoint: "localhost:9000", Bucket: "test"},
				Gateway:  GatewayConfig{Timeout: 30, MaxRetries: 3},
			},
			wantErr: true,
			errMsg:  "database.dbname",
		},
		{
			name: "release mode valid",
			cfg: Config{
				Server:   ServerConfig{Port: 8080, Mode: "release"},
				Database: DatabaseConfig{Host: "localhost", Port: 5432, DBName: "test"},
				Redis:    RedisConfig{Host: "localhost", Port: 6379},
				Minio:    MinioConfig{Endpoint: "localhost:9000", Bucket: "test"},
				Gateway:  GatewayConfig{Timeout: 30, MaxRetries: 3},
			},
			wantErr: false,
		},
		{
			name: "test mode valid",
			cfg: Config{
				Server:   ServerConfig{Port: 8080, Mode: "test"},
				Database: DatabaseConfig{Host: "localhost", Port: 5432, DBName: "test"},
				Redis:    RedisConfig{Host: "localhost", Port: 6379},
				Minio:    MinioConfig{Endpoint: "localhost:9000", Bucket: "test"},
				Gateway:  GatewayConfig{Timeout: 30, MaxRetries: 3},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !containsString(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestValidateGatewayConfig 测试 Gateway 配置验证
func TestValidateGatewayConfig(t *testing.T) {
	baseConfig := func() Config {
		return Config{
			Server:   ServerConfig{Port: 8080, Mode: "debug"},
			Database: DatabaseConfig{Host: "localhost", Port: 5432, DBName: "test"},
			Redis:    RedisConfig{Host: "localhost", Port: 6379},
			Minio:    MinioConfig{Endpoint: "localhost:9000", Bucket: "test"},
			Gateway:  GatewayConfig{Timeout: 30, MaxRetries: 3},
		}
	}

	t.Run("invalid timeout - zero", func(t *testing.T) {
		cfg := baseConfig()
		cfg.Gateway.Timeout = 0
		err := cfg.Validate()
		if err == nil || !containsString(err.Error(), "gateway.timeout") {
			t.Errorf("expected gateway.timeout error")
		}
	})

	t.Run("invalid max_retries - negative", func(t *testing.T) {
		cfg := baseConfig()
		cfg.Gateway.MaxRetries = -1
		err := cfg.Validate()
		if err == nil || !containsString(err.Error(), "gateway.max_retries") {
			t.Errorf("expected gateway.max_retries error")
		}
	})
}

// TestValidateMinioConfig 测试 MinIO 配置验证
func TestValidateMinioConfig(t *testing.T) {
	baseConfig := func() Config {
		return Config{
			Server:   ServerConfig{Port: 8080, Mode: "debug"},
			Database: DatabaseConfig{Host: "localhost", Port: 5432, DBName: "test"},
			Redis:    RedisConfig{Host: "localhost", Port: 6379},
			Minio:    MinioConfig{Endpoint: "localhost:9000", Bucket: "test"},
			Gateway:  GatewayConfig{Timeout: 30, MaxRetries: 3},
		}
	}

	t.Run("missing endpoint", func(t *testing.T) {
		cfg := baseConfig()
		cfg.Minio.Endpoint = ""
		err := cfg.Validate()
		if err == nil || !containsString(err.Error(), "minio.endpoint") {
			t.Errorf("expected minio.endpoint error")
		}
	})

	t.Run("missing bucket", func(t *testing.T) {
		cfg := baseConfig()
		cfg.Minio.Bucket = ""
		err := cfg.Validate()
		if err == nil || !containsString(err.Error(), "minio.bucket") {
			t.Errorf("expected minio.bucket error")
		}
	})
}

// TestValidateBlockchainConfig 测试区块链配置验证
func TestValidateBlockchainConfig(t *testing.T) {
	baseConfig := func() Config {
		return Config{
			Server:   ServerConfig{Port: 8080, Mode: "debug"},
			Database: DatabaseConfig{Host: "localhost", Port: 5432, DBName: "test"},
			Redis:    RedisConfig{Host: "localhost", Port: 6379},
			Minio:    MinioConfig{Endpoint: "localhost:9000", Bucket: "test"},
			Gateway:  GatewayConfig{Timeout: 30, MaxRetries: 3},
			Features: FeatureConfig{BlockchainAnchor: true},
		}
	}

	t.Run("ethereum enabled without rpc_url", func(t *testing.T) {
		cfg := baseConfig()
		cfg.Anchor.Ethereum.Enabled = true
		cfg.Anchor.Ethereum.RPCURL = ""
		err := cfg.Validate()
		if err == nil || !containsString(err.Error(), "anchor.ethereum.rpc_url") {
			t.Errorf("expected anchor.ethereum.rpc_url error")
		}
	})

	t.Run("ethereum enabled with rpc_url", func(t *testing.T) {
		cfg := baseConfig()
		cfg.Anchor.Ethereum.Enabled = true
		cfg.Anchor.Ethereum.RPCURL = "https://mainnet.infura.io/v3/xxx"
		err := cfg.Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// TestValidationErrors 测试验证错误格式
func TestValidationErrors(t *testing.T) {
	t.Run("single error", func(t *testing.T) {
		err := ValidationError{Field: "test.field", Message: "is invalid"}
		if err.Error() != "test.field: is invalid" {
			t.Errorf("unexpected error format: %s", err.Error())
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		errs := ValidationErrors{
			{Field: "field1", Message: "error1"},
			{Field: "field2", Message: "error2"},
		}
		errStr := errs.Error()
		if !containsString(errStr, "field1") || !containsString(errStr, "field2") {
			t.Errorf("expected both errors in output: %s", errStr)
		}
	})

	t.Run("empty errors", func(t *testing.T) {
		errs := ValidationErrors{}
		if errs.Error() != "" {
			t.Errorf("empty errors should return empty string")
		}
	})
}

// containsString checks if s contains substr
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
