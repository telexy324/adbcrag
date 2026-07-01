package util

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var allowedExt = map[string]bool{
	".md":   true,
	".txt":  true,
	".docx": true,
	".xlsx": true,
}

const SupportedUploadTypes = ".md, .txt, .docx, .xlsx"

func ValidateUpload(fileHeader *multipart.FileHeader, maxMB int) (string, error) {
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedExt[ext] {
		return "", fmt.Errorf("unsupported file type: %s, supported types: %s", ext, SupportedUploadTypes)
	}
	if fileHeader.Size > int64(maxMB)*1024*1024 {
		return "", fmt.Errorf("file size exceeds %dMB", maxMB)
	}
	return ext, nil
}

func ValidateUploadContent(file multipart.File, ext string) error {
	if ext != ".docx" && ext != ".xlsx" {
		return nil
	}
	header := make([]byte, 4)
	n, err := io.ReadFull(file, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		return err
	}
	if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil {
		return seekErr
	}
	if n < 4 || string(header[:2]) != "PK" {
		return fmt.Errorf("%s is not a valid Office Open XML file; please upload a real %s file, not .doc/.xls renamed to %s or an encrypted/corrupted file", ext, ext, ext)
	}
	return nil
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
