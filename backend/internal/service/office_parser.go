package service

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"ops-kb-rag/backend/internal/util"

	"github.com/extrame/xls"
	docx "github.com/fumiama/go-docx"
	"github.com/xuri/excelize/v2"
)

func ParseOfficeFile(filePath string) (string, error) {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".docx":
		return ParseDOCX(filePath)
	case ".xlsx":
		return ParseXLSX(filePath)
	case ".xls":
		return ParseXLS(filePath)
	case ".doc":
		return "", fmt.Errorf("legacy .doc parsing is not supported by the Go library parser; please upload .docx or convert the file before uploading")
	default:
		return "", fmt.Errorf("office parser for %s is not implemented", filepath.Ext(filePath))
	}
}

func ParseDOCX(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	doc, err := docx.Parse(file, info.Size())
	if err != nil {
		if isZipChecksumError(err) {
			return ParseDOCXLenient(filePath)
		}
		return "", friendlyOfficeParseError(".docx", err)
	}

	var parts []string
	for _, item := range doc.Document.Body.Items {
		switch typed := item.(type) {
		case *docx.Paragraph:
			if text := strings.TrimSpace(typed.String()); text != "" {
				parts = append(parts, text)
			}
		case *docx.Table:
			if text := strings.TrimSpace(typed.String()); text != "" {
				parts = append(parts, text)
			}
		}
	}
	return normalizedOfficeText(strings.Join(parts, "\n\n"), filePath)
}

func ParseDOCXLenient(filePath string) (string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return "", friendlyOfficeParseError(".docx", err)
	}
	defer reader.Close()

	file := findZipFile(reader.File, "word/document.xml")
	if file == nil {
		return "", fmt.Errorf("invalid .docx file: missing word/document.xml")
	}
	rc, err := file.Open()
	if err != nil {
		return "", friendlyOfficeParseError(".docx", err)
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	var paragraphs []string
	var current strings.Builder
	inText := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			if isZipChecksumError(err) && (len(paragraphs) > 0 || strings.TrimSpace(current.String()) != "") {
				break
			}
			return "", friendlyOfficeParseError(".docx", err)
		}
		switch item := token.(type) {
		case xml.StartElement:
			switch item.Name.Local {
			case "t":
				inText = true
			case "tab":
				current.WriteString("\t")
			case "br", "cr":
				current.WriteString("\n")
			}
		case xml.CharData:
			if inText {
				current.Write([]byte(item))
			}
		case xml.EndElement:
			switch item.Name.Local {
			case "t":
				inText = false
			case "p":
				text := strings.TrimSpace(current.String())
				if text != "" {
					paragraphs = append(paragraphs, text)
				}
				current.Reset()
			}
		}
	}
	if tail := strings.TrimSpace(current.String()); tail != "" {
		paragraphs = append(paragraphs, tail)
	}
	return normalizedOfficeText(strings.Join(paragraphs, "\n\n"), filePath)
}

func ParseXLSX(filePath string) (string, error) {
	file, err := excelize.OpenFile(filePath)
	if err != nil {
		return "", friendlyOfficeParseError(".xlsx", err)
	}
	defer file.Close()

	var doc strings.Builder
	for _, sheetName := range file.GetSheetList() {
		rows, err := file.GetRows(sheetName)
		if err != nil {
			return "", friendlyOfficeParseError(".xlsx", err)
		}
		writeSheetText(&doc, sheetName, rows)
	}
	return normalizedOfficeText(doc.String(), filePath)
}

func ParseXLS(filePath string) (string, error) {
	workbook, err := xls.Open(filePath, "utf-8")
	if err != nil {
		return "", friendlyOfficeParseError(".xls", err)
	}

	var doc strings.Builder
	for i := 0; i < workbook.NumSheets(); i++ {
		sheet := workbook.GetSheet(i)
		if sheet == nil {
			continue
		}
		var rows [][]string
		for rowIndex := 0; rowIndex <= int(sheet.MaxRow); rowIndex++ {
			row := safeXLSRow(sheet, rowIndex)
			if row == nil {
				continue
			}
			var cells []string
			for colIndex := row.FirstCol(); colIndex < row.LastCol(); colIndex++ {
				cells = append(cells, row.Col(colIndex))
			}
			rows = append(rows, cells)
		}
		writeSheetText(&doc, sheet.Name, rows)
	}
	return normalizedOfficeText(doc.String(), filePath)
}

func safeXLSRow(sheet *xls.WorkSheet, rowIndex int) (row *xls.Row) {
	defer func() {
		if recover() != nil {
			row = nil
		}
	}()
	return sheet.Row(rowIndex)
}

func writeSheetText(doc *strings.Builder, sheetName string, rows [][]string) {
	var renderedRows []string
	for _, row := range rows {
		rendered := strings.TrimSpace(strings.Join(trimTrailingEmptyCells(row), "\t"))
		if rendered != "" {
			renderedRows = append(renderedRows, rendered)
		}
	}
	if len(renderedRows) == 0 {
		return
	}
	if doc.Len() > 0 {
		doc.WriteString("\n\n")
	}
	doc.WriteString("# ")
	doc.WriteString(sheetName)
	doc.WriteString("\n\n")
	doc.WriteString(strings.Join(renderedRows, "\n"))
}

func trimTrailingEmptyCells(cells []string) []string {
	end := len(cells)
	for end > 0 && strings.TrimSpace(cells[end-1]) == "" {
		end--
	}
	return cells[:end]
}

func normalizedOfficeText(text, filePath string) (string, error) {
	text = util.NormalizeText(text)
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("office parser produced empty text for %s", filepath.Base(filePath))
	}
	return text, nil
}

func findZipFile(files []*zip.File, name string) *zip.File {
	name = strings.TrimPrefix(path.Clean(name), "/")
	for _, file := range files {
		if path.Clean(file.Name) == name {
			return file
		}
	}
	return nil
}

func friendlyOfficeParseError(ext string, err error) error {
	if isZipChecksumError(err) {
		return fmt.Errorf("%s file appears to be corrupted or incompletely uploaded: %w", ext, err)
	}
	if errors.Is(err, zip.ErrFormat) {
		return fmt.Errorf("%s file is not a valid Office document; please upload a real %s file, not a renamed or encrypted/corrupted file", ext, ext)
	}
	return err
}

func isZipChecksumError(err error) bool {
	return errors.Is(err, zip.ErrChecksum) || strings.Contains(strings.ToLower(err.Error()), "checksum error")
}
