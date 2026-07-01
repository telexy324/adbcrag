package service

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"

	"ops-kb-rag/backend/internal/util"
)

func ParseDOCX(filePath string) (string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return "", friendlyOfficeZipError(".docx", err)
	}
	defer reader.Close()

	file := findZipFile(reader.File, "word/document.xml")
	if file == nil {
		return "", fmt.Errorf("docx missing word/document.xml")
	}
	rc, err := file.Open()
	if err != nil {
		return "", err
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
			return "", err
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
	return util.NormalizeText(strings.Join(paragraphs, "\n\n")), nil
}

func ParseXLSX(filePath string) (string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return "", friendlyOfficeZipError(".xlsx", err)
	}
	defer reader.Close()

	sharedStrings, err := parseSharedStrings(reader.File)
	if err != nil {
		return "", err
	}
	sheets := parseWorkbookSheets(reader.File)
	if len(sheets) == 0 {
		sheets = fallbackWorksheetFiles(reader.File)
	}
	if len(sheets) == 0 {
		return "", fmt.Errorf("xlsx has no worksheets")
	}

	var doc strings.Builder
	for _, sheet := range sheets {
		rows, err := parseWorksheet(reader.File, sheet.Path, sharedStrings)
		if err != nil {
			return "", err
		}
		if len(rows) == 0 {
			continue
		}
		if doc.Len() > 0 {
			doc.WriteString("\n\n")
		}
		doc.WriteString("# ")
		doc.WriteString(sheet.Name)
		doc.WriteString("\n\n")
		doc.WriteString(strings.Join(rows, "\n"))
	}
	return util.NormalizeText(doc.String()), nil
}

type workbookSheet struct {
	Name string
	Path string
}

func parseSharedStrings(files []*zip.File) ([]string, error) {
	file := findZipFile(files, "xl/sharedStrings.xml")
	if file == nil {
		return nil, nil
	}
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	var values []string
	var current strings.Builder
	inSI := false
	inText := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch item := token.(type) {
		case xml.StartElement:
			if item.Name.Local == "si" {
				inSI = true
				current.Reset()
			}
			if inSI && item.Name.Local == "t" {
				inText = true
			}
		case xml.CharData:
			if inSI && inText {
				current.Write([]byte(item))
			}
		case xml.EndElement:
			if item.Name.Local == "t" {
				inText = false
			}
			if item.Name.Local == "si" {
				values = append(values, current.String())
				inSI = false
			}
		}
	}
	return values, nil
}

func parseWorkbookSheets(files []*zip.File) []workbookSheet {
	rels := parseWorkbookRelationships(files)
	file := findZipFile(files, "xl/workbook.xml")
	if file == nil {
		return nil
	}
	rc, err := file.Open()
	if err != nil {
		return nil
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	var sheets []workbookSheet
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "sheet" {
			continue
		}
		name := attrValue(start, "name")
		if name == "" {
			name = fmt.Sprintf("Sheet%d", len(sheets)+1)
		}
		relID := attrValue(start, "id")
		target := rels[relID]
		if target == "" {
			continue
		}
		sheets = append(sheets, workbookSheet{Name: name, Path: resolveWorkbookTarget(target)})
	}
	return sheets
}

func parseWorkbookRelationships(files []*zip.File) map[string]string {
	result := map[string]string{}
	file := findZipFile(files, "xl/_rels/workbook.xml.rels")
	if file == nil {
		return result
	}
	rc, err := file.Open()
	if err != nil {
		return result
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return result
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "Relationship" {
			continue
		}
		id := attrValue(start, "Id")
		target := attrValue(start, "Target")
		if id != "" && target != "" {
			result[id] = target
		}
	}
	return result
}

func fallbackWorksheetFiles(files []*zip.File) []workbookSheet {
	var names []string
	for _, file := range files {
		if strings.HasPrefix(file.Name, "xl/worksheets/") && strings.HasSuffix(file.Name, ".xml") {
			names = append(names, file.Name)
		}
	}
	sort.Strings(names)
	sheets := make([]workbookSheet, 0, len(names))
	for i, name := range names {
		sheets = append(sheets, workbookSheet{Name: fmt.Sprintf("Sheet%d", i+1), Path: name})
	}
	return sheets
}

func parseWorksheet(files []*zip.File, sheetPath string, sharedStrings []string) ([]string, error) {
	file := findZipFile(files, sheetPath)
	if file == nil {
		return nil, fmt.Errorf("xlsx missing worksheet %s", sheetPath)
	}
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	var rows []string
	var cells []string
	var current strings.Builder
	var cellType string
	inValue := false
	inInlineText := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch item := token.(type) {
		case xml.StartElement:
			switch item.Name.Local {
			case "row":
				cells = nil
			case "c":
				cellType = attrValue(item, "t")
				current.Reset()
			case "v":
				inValue = true
			case "t":
				if cellType == "inlineStr" || cellType == "str" {
					inInlineText = true
				}
			}
		case xml.CharData:
			if inValue || inInlineText {
				current.Write([]byte(item))
			}
		case xml.EndElement:
			switch item.Name.Local {
			case "v":
				inValue = false
			case "t":
				inInlineText = false
			case "c":
				cells = append(cells, resolveCellValue(strings.TrimSpace(current.String()), cellType, sharedStrings))
				current.Reset()
			case "row":
				row := strings.TrimSpace(strings.Join(trimTrailingEmpty(cells), "\t"))
				if row != "" {
					rows = append(rows, row)
				}
			}
		}
	}
	return rows, nil
}

func resolveCellValue(raw, cellType string, sharedStrings []string) string {
	if raw == "" {
		return ""
	}
	if cellType == "s" {
		index, err := strconv.Atoi(raw)
		if err == nil && index >= 0 && index < len(sharedStrings) {
			return sharedStrings[index]
		}
	}
	return raw
}

func trimTrailingEmpty(values []string) []string {
	end := len(values)
	for end > 0 && strings.TrimSpace(values[end-1]) == "" {
		end--
	}
	return values[:end]
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

func resolveWorkbookTarget(target string) string {
	target = strings.TrimPrefix(target, "/")
	if strings.HasPrefix(target, "xl/") {
		return path.Clean(target)
	}
	return path.Clean("xl/" + target)
}

func attrValue(start xml.StartElement, localName string) string {
	for _, attr := range start.Attr {
		if attr.Name.Local == localName {
			return attr.Value
		}
	}
	return ""
}

func friendlyOfficeZipError(ext string, err error) error {
	if errors.Is(err, zip.ErrFormat) {
		return fmt.Errorf("%s is not a valid Office Open XML file; please upload a real %s file, not .doc/.xls renamed to %s or an encrypted/corrupted file", ext, ext, ext)
	}
	return err
}
