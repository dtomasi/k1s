package extractor

import "testing"

func TestExtractor_Extract(t *testing.T) {
	extractor := NewExtractor()

	err := extractor.Extract([]string{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
