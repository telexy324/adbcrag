package service

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	docx "github.com/fumiama/go-docx"
	"github.com/xuri/excelize/v2"
)

func TestParseDOCXWithGoLibrary(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "manual.docx")
	doc := docx.New().WithDefaultTheme()
	doc.AddParagraph().AddText("Redis 内存告警处置手册")
	doc.AddParagraph().AddText("执行 info memory 检查 used_memory。")

	file, err := os.Create(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doc.WriteTo(file); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	text, err := ParseDOCX(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Redis 内存告警处置手册") || !strings.Contains(text, "info memory") {
		t.Fatalf("unexpected docx text: %s", text)
	}
}

func TestParseDOCXLenientIgnoresChecksumAfterText(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "manual.docx")
	writeStoredDOCX(t, filePath, `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>Redis 内存告警处置手册</w:t></w:r></w:p>
    <w:p><w:r><w:t>执行 info memory 检查 used_memory。</w:t></w:r></w:p>
  </w:body>
</w:document>`)
	corruptStoredZipPayload(t, filePath, "used_memory", "max_memory")

	text, err := ParseDOCXLenient(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Redis 内存告警处置手册") || !strings.Contains(text, "max_memory") {
		t.Fatalf("unexpected lenient docx text: %s", text)
	}
}

func TestParseXLSXWithGoLibrary(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "manual.xlsx")
	file := excelize.NewFile()
	sheetName := "告警处置"
	file.SetSheetName("Sheet1", sheetName)
	if err := file.SetCellValue(sheetName, "A1", "组件"); err != nil {
		t.Fatal(err)
	}
	if err := file.SetCellValue(sheetName, "B1", "Redis"); err != nil {
		t.Fatal(err)
	}
	if err := file.SetCellValue(sheetName, "A2", "处理建议"); err != nil {
		t.Fatal(err)
	}
	if err := file.SetCellValue(sheetName, "B2", "检查 bigkeys"); err != nil {
		t.Fatal(err)
	}
	if err := file.SaveAs(filePath); err != nil {
		t.Fatal(err)
	}

	text, err := ParseXLSX(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "# 告警处置") || !strings.Contains(text, "Redis") || !strings.Contains(text, "检查 bigkeys") {
		t.Fatalf("unexpected xlsx text: %s", text)
	}
}

func writeStoredDOCX(t *testing.T, filePath, documentXML string) {
	t.Helper()
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	header := &zip.FileHeader{Name: "word/document.xml", Method: zip.Store}
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(documentXML)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
}

func corruptStoredZipPayload(t *testing.T, filePath, oldValue, newValue string) {
	t.Helper()
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	oldBytes := []byte(oldValue)
	index := strings.Index(string(data), oldValue)
	if index < 0 {
		t.Fatalf("could not find %q in zip payload", oldValue)
	}
	copy(data[index:index+len(oldBytes)], []byte(newValue))
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestParseOfficeFileRejectsLegacyDoc(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "manual.doc")
	if err := os.WriteFile(filePath, []byte("legacy word payload"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ParseOfficeFile(filePath)
	if err == nil {
		t.Fatal("expected legacy doc error")
	}
	if !strings.Contains(err.Error(), "legacy .doc parsing is not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}
