package util

import (
	"mime/multipart"
	"strings"
	"testing"
)

type fakeMultipartFile struct {
	*strings.Reader
}

func (f fakeMultipartFile) Close() error { return nil }

func TestValidateUploadSupportsOfficeFiles(t *testing.T) {
	for _, name := range []string{"manual.md", "manual.txt", "manual.doc", "manual.docx", "manual.xls", "manual.xlsx", "MANUAL.DOCX"} {
		t.Run(name, func(t *testing.T) {
			ext, err := ValidateUpload(&multipart.FileHeader{Filename: name, Size: 1024}, 50)
			if err != nil {
				t.Fatal(err)
			}
			if ext == "" {
				t.Fatal("expected extension")
			}
		})
	}
}

func TestValidateUploadContentDefersOfficeValidationToParser(t *testing.T) {
	err := ValidateUploadContent(fakeMultipartFile{Reader: strings.NewReader("not a zip")}, ".docx")
	if err != nil {
		t.Fatal(err)
	}
}
