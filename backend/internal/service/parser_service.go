package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ops-kb-rag/backend/internal/util"
)

type ParserService struct{}

func NewParserService() *ParserService {
	return &ParserService{}
}

func (s *ParserService) Parse(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".txt":
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return util.NormalizeText(string(data)), nil
	default:
		return "", fmt.Errorf("parser for %s is not implemented", ext)
	}
}
