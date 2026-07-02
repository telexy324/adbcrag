package handler

import (
	"errors"
	"testing"
)

func TestUploadErrorStatusTreatsOfficeChecksumAsBadRequest(t *testing.T) {
	status := uploadErrorStatus(errors.New("zip: checksum error"))
	if status != 400 {
		t.Fatalf("expected 400, got %d", status)
	}
}
