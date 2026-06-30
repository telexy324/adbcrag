package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv           string
	AppPort          string
	DBHost           string
	DBPort           string
	DBUser           string
	DBPassword       string
	DBName           string
	DBSSLMode        string
	FileStorageType  string
	LocalFileDir     string
	MaxUploadMB      int
	DeepSeekBaseURL  string
	DeepSeekAPIKey   string
	DeepSeekModel    string
	EmbeddingBaseURL string
	EmbeddingAPIKey  string
	EmbeddingModel   string
	EmbeddingDim     int
	RAGTopK          int
	RAGMinScore      float64
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		AppEnv:           getEnv("APP_ENV", "dev"),
		AppPort:          getEnv("APP_PORT", "8080"),
		DBHost:           getEnv("DB_HOST", "127.0.0.1"),
		DBPort:           getEnv("DB_PORT", "5432"),
		DBUser:           getEnv("DB_USER", "postgres"),
		DBPassword:       getEnv("DB_PASSWORD", "postgres"),
		DBName:           getEnv("DB_NAME", "ops_kb"),
		DBSSLMode:        getEnv("DB_SSLMODE", "disable"),
		FileStorageType:  getEnv("FILE_STORAGE_TYPE", "local"),
		LocalFileDir:     getEnv("LOCAL_FILE_DIR", "./data/uploads"),
		MaxUploadMB:      getEnvInt("MAX_UPLOAD_MB", 50),
		DeepSeekBaseURL:  getEnv("DEEPSEEK_BASE_URL", "http://deepseek-v4.internal.local/v1"),
		DeepSeekAPIKey:   getEnv("DEEPSEEK_API_KEY", "local-key"),
		DeepSeekModel:    getEnv("DEEPSEEK_MODEL", "deepseek-v4"),
		EmbeddingBaseURL: getEnv("EMBEDDING_BASE_URL", "http://embedding.internal.local/v1"),
		EmbeddingAPIKey:  getEnv("EMBEDDING_API_KEY", "local-key"),
		EmbeddingModel:   getEnv("EMBEDDING_MODEL", "bge-m3"),
		EmbeddingDim:     getEnvInt("EMBEDDING_DIM", 1024),
		RAGTopK:          getEnvInt("RAG_TOP_K", 5),
		RAGMinScore:      getEnvFloat("RAG_MIN_SCORE", 0.3),
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

func getEnvFloat(key string, fallback float64) float64 {
	value, err := strconv.ParseFloat(getEnv(key, ""), 64)
	if err != nil {
		return fallback
	}
	return value
}
