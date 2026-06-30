package util

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var allowedExt = map[string]bool{".md": true, ".txt": true}

func ValidateUpload(fileHeader *multipart.FileHeader, maxMB int) (string, error) {
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedExt[ext] {
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
	if fileHeader.Size > int64(maxMB)*1024*1024 {
		return "", fmt.Errorf("file size exceeds %dMB", maxMB)
	}
	return ext, nil
}

func SaveUploadedFile(src multipart.File, fileHeader *multipart.FileHeader, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(fileHeader.Filename))
	dstPath := filepath.Join(dir, name)
	dst, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := dst.ReadFrom(src); err != nil {
		return "", err
	}
	return dstPath, nil
}
