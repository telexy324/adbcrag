package service

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDOCX(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "manual.docx")
	writeZip(t, filePath, map[string]string{
		"word/document.xml": `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>Redis 内存告警处置手册</w:t></w:r></w:p>
    <w:p><w:r><w:t>执行 info memory 检查 used_memory。</w:t></w:r></w:p>
  </w:body>
</w:document>`,
	})

	text, err := ParseDOCX(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Redis 内存告警处置手册") || !strings.Contains(text, "info memory") {
		t.Fatalf("unexpected docx text: %s", text)
	}
}

func TestParseXLSX(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "manual.xlsx")
	writeZip(t, filePath, map[string]string{
		"xl/workbook.xml": `<?xml version="1.0" encoding="UTF-8"?>
<workbook xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets><sheet name="告警处置" sheetId="1" r:id="rId1"/></sheets>
</workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Target="worksheets/sheet1.xml"/>
</Relationships>`,
		"xl/sharedStrings.xml": `<?xml version="1.0" encoding="UTF-8"?>
<sst><si><t>组件</t></si><si><t>Redis</t></si><si><t>处理建议</t></si><si><t>检查 bigkeys</t></si></sst>`,
		"xl/worksheets/sheet1.xml": `<?xml version="1.0" encoding="UTF-8"?>
<worksheet>
  <sheetData>
    <row><c t="s"><v>0</v></c><c t="s"><v>1</v></c></row>
    <row><c t="s"><v>2</v></c><c t="s"><v>3</v></c></row>
  </sheetData>
</worksheet>`,
	})

	text, err := ParseXLSX(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "# 告警处置") || !strings.Contains(text, "Redis") || !strings.Contains(text, "检查 bigkeys") {
		t.Fatalf("unexpected xlsx text: %s", text)
	}
}

func writeZip(t *testing.T, filePath string, files map[string]string) {
	t.Helper()
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	defer writer.Close()
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
}
