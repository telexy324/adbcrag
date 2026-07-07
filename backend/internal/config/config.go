package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv          string
	AppPort         string
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	DBSSLMode       string
	FileStorageType string
	LocalFileDir    string
	MaxUploadMB     int
	DeepSeekBaseURL string
	DeepSeekAPIKey  string
	DeepSeekModel   string
	RAGTopK         int
	RAGRecallK      int
	CredentialKey   string
	LogMaxLines     int
	LogMaxBytes     int
	LogTimeoutSec   int
	SSHTimeoutSec   int
	ESTimeoutSec    int
	K8sLogTailLines int
	K8sLogMaxBytes  int
	InitAdminUser   string
	InitAdminPass   string
	JWTSecret       string
	JWTExpireHours  int
	LogLevel        string
	LogFormat       string
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		AppEnv:          getEnv("APP_ENV", "dev"),
		AppPort:         getEnv("APP_PORT", "8080"),
		DBHost:          getEnv("DB_HOST", "127.0.0.1"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBUser:          getEnv("DB_USER", "postgres"),
		DBPassword:      getEnv("DB_PASSWORD", "postgres"),
		DBName:          getEnv("DB_NAME", "ops_kb"),
		DBSSLMode:       getEnv("DB_SSLMODE", "disable"),
		FileStorageType: getEnv("FILE_STORAGE_TYPE", "local"),
		LocalFileDir:    getEnv("LOCAL_FILE_DIR", "./data/uploads"),
		MaxUploadMB:     getEnvInt("MAX_UPLOAD_MB", 50),
		DeepSeekBaseURL: getEnv("DEEPSEEK_BASE_URL", "http://deepseek-v4.internal.local/v1"),
		DeepSeekAPIKey:  getEnv("DEEPSEEK_API_KEY", "local-key"),
		DeepSeekModel:   getEnv("DEEPSEEK_MODEL", "deepseek-v4"),
		RAGTopK:         getEnvInt("RAG_TOP_K", 5),
		RAGRecallK:      getEnvInt("RAG_RECALL_K", 30),
		CredentialKey:   getEnv("CREDENTIAL_ENCRYPTION_KEY", "dev-only-change-me-32-bytes-minimum"),
		LogMaxLines:     getEnvInt("LOG_SAMPLE_MAX_LINES", 500),
		LogMaxBytes:     getEnvInt("LOG_SAMPLE_MAX_BYTES", 262144),
		LogTimeoutSec:   getEnvInt("LOG_ANALYSIS_TIMEOUT_SECONDS", 60),
		SSHTimeoutSec:   getEnvInt("SSH_CONNECT_TIMEOUT_SECONDS", 10),
		ESTimeoutSec:    getEnvInt("ES_QUERY_TIMEOUT_SECONDS", 15),
		K8sLogTailLines: getEnvInt("K8S_LOG_TAIL_LINES", 300),
		K8sLogMaxBytes:  getEnvInt("K8S_LOG_MAX_BYTES", 262144),
		InitAdminUser:   getEnv("INIT_ADMIN_USERNAME", "admin"),
		InitAdminPass:   getEnv("INIT_ADMIN_PASSWORD", "Admin@123456"),
		JWTSecret:       getEnv("JWT_SECRET", "dev-only-change-me-jwt-secret"),
		JWTExpireHours:  getEnvInt("JWT_EXPIRE_HOURS", 24),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		LogFormat:       getEnv("LOG_FORMAT", "json"),
	}
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value, err := strconv.Atoi(getEnv(key, ""))
	if err != nil {
		return fallback
	}
	return value
}
