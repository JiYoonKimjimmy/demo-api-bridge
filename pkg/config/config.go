package config

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config는 애플리케이션의 전체 설정을 나타냅니다.
type Config struct {
	Server         ServerConfig         `yaml:"server"`
	Log            LogConfig            `yaml:"log"`
	Database       DatabaseConfig       `yaml:"database"`
	Redis          RedisConfig          `yaml:"redis"`
	ExternalAPI    ExternalAPIConfig    `yaml:"external_api"`
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
	Metrics        MetricsConfig        `yaml:"metrics"`
	Cache          CacheConfig          `yaml:"cache"`
}

// ServerConfig는 서버 관련 설정을 나타냅니다.
type ServerConfig struct {
	Port           string        `yaml:"port"`
	Mode           string        `yaml:"mode"`
	ReadTimeout    time.Duration `yaml:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout"`
	MaxHeaderBytes int           `yaml:"max_header_bytes"`
}

// LogConfig는 로깅 관련 설정을 나타냅니다.
type LogConfig struct {
	Level    string `yaml:"level"`
	Format   string `yaml:"format"`
	Output   string `yaml:"output"`
	FilePath string `yaml:"file_path"`
}

// DatabaseConfig는 OracleDB 관련 설정을 나타냅니다.
type DatabaseConfig struct {
	Host              string        `yaml:"host"`
	Port              int           `yaml:"port"`
	SID               string        `yaml:"sid"`
	Username          string        `yaml:"username"`
	Password          string        `yaml:"password"`
	MaxOpenConns      int           `yaml:"max_open_conns"`
	MaxIdleConns      int           `yaml:"max_idle_conns"`
	ConnMaxLifetime   time.Duration `yaml:"conn_max_lifetime"`
	ConnectionTimeout time.Duration `yaml:"connection_timeout"`
}

// RedisConfig는 Redis 관련 설정을 나타냅니다.
type RedisConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	Password     string        `yaml:"password"`
	DB           int           `yaml:"db"`
	PoolSize     int           `yaml:"pool_size"`
	MinIdleConns int           `yaml:"min_idle_conns"`
	DialTimeout  time.Duration `yaml:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// ExternalAPIConfig는 외부 API 호출 관련 설정을 나타냅니다.
type ExternalAPIConfig struct {
	BaseURL                string        `yaml:"base_url"`
	Timeout                time.Duration `yaml:"timeout"`
	RetryCount             int           `yaml:"retry_count"`
	RetryDelay             time.Duration `yaml:"retry_delay"`
	MaxRetryDelay          time.Duration `yaml:"max_retry_delay"`
	RetryBackoffMultiplier float64       `yaml:"retry_backoff_multiplier"`
}

// CircuitBreakerConfig는 Circuit Breaker 관련 설정을 나타냅니다.
type CircuitBreakerConfig struct {
	MaxRequests uint32        `yaml:"max_requests"`
	Interval    time.Duration `yaml:"interval"`
	Timeout     time.Duration `yaml:"timeout"`
}

// MetricsConfig는 메트릭 관련 설정을 나타냅니다.
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	Path    string `yaml:"path"`
}

// CacheConfig는 캐시 관련 설정을 나타냅니다.
type CacheConfig struct {
	DefaultTTL      time.Duration `yaml:"default_ttl"`
	RoutingRulesTTL time.Duration `yaml:"routing_rules_ttl"`
	APIResponseTTL  time.Duration `yaml:"api_response_ttl"`
}

// LoadConfig는 설정 파일을 로드합니다.
func LoadConfig(configPath string) (*Config, error) {
	// 설정 파일이 없으면 기본값으로 초기화
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return getDefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 환경변수로 오버라이드
	overrideFromEnv(&config)

	return &config, nil
}

// getDefaultConfig는 기본 설정을 반환합니다.
func getDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:           "10019",
			Mode:           "release",
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1048576,
		},
		Log: LogConfig{
			Level:    "info",
			Format:   "json",
			Output:   "stdout",
			FilePath: "./logs/app.log",
		},
		Database: DatabaseConfig{
			Host:              "localhost",
			Port:              1521,
			SID:               "ORCL",
			MaxOpenConns:      25,
			MaxIdleConns:      5,
			ConnMaxLifetime:   5 * time.Minute,
			ConnectionTimeout: 10 * time.Second,
		},
		Redis: RedisConfig{
			Host:         "localhost",
			Port:         6379,
			DB:           0,
			PoolSize:     10,
			MinIdleConns: 5,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		},
		ExternalAPI: ExternalAPIConfig{
			BaseURL:                "https://api.example.com",
			Timeout:                30 * time.Second,
			RetryCount:             3,
			RetryDelay:             1 * time.Second,
			MaxRetryDelay:          10 * time.Second,
			RetryBackoffMultiplier: 2.0,
		},
		CircuitBreaker: CircuitBreakerConfig{
			MaxRequests: 5,
			Interval:    10 * time.Second,
			Timeout:     5 * time.Second,
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Port:    9090,
			Path:    "/metrics",
		},
		Cache: CacheConfig{
			DefaultTTL:      300 * time.Second,
			RoutingRulesTTL: 3600 * time.Second,
			APIResponseTTL:  600 * time.Second,
		},
	}
}

// overrideFromEnv는 환경변수로 설정을 오버라이드합니다.
func overrideFromEnv(config *Config) {
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		config.Database.Host = dbHost
	}
	if dbPort := os.Getenv("DB_PORT"); dbPort != "" {
		config.Database.Port = parseInt(dbPort)
	}
	if dbUser := os.Getenv("DB_USERNAME"); dbUser != "" {
		config.Database.Username = dbUser
	}
	if dbPass := os.Getenv("DB_PASSWORD"); dbPass != "" {
		config.Database.Password = dbPass
	}
	if dbSID := os.Getenv("DB_SID"); dbSID != "" {
		config.Database.SID = dbSID
	}

	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		config.Redis.Host = redisHost
	}
	if redisPort := os.Getenv("REDIS_PORT"); redisPort != "" {
		config.Redis.Port = parseInt(redisPort)
	}
	if redisPass := os.Getenv("REDIS_PASSWORD"); redisPass != "" {
		config.Redis.Password = redisPass
	}

	if serverPort := os.Getenv("SERVER_PORT"); serverPort != "" {
		config.Server.Port = serverPort
	}
}

// parseInt는 문자열을 정수로 변환합니다.
func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

// GetDSN은 OracleDB 연결 문자열을 반환합니다 (sijms/go-ora/v2 형식).
func (d *DatabaseConfig) GetDSN() string {
	// 비밀번호를 URL 인코딩하여 특수문자 문제를 해결
	encodedPassword := url.QueryEscape(d.Password)
	return fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		d.Username,
		encodedPassword,
		d.Host,
		d.Port,
		d.SID,
	)
}

// GetOracleDSN은 Go Oracle 드라이버용 연결 문자열을 반환합니다.
func (d *DatabaseConfig) GetOracleDSN() string {
	return fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		d.Username,
		d.Password,
		d.Host,
		d.Port,
		d.SID,
	)
}

// GetRedisAddr은 Redis 주소를 반환합니다.
func (r *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}
