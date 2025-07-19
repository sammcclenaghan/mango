package converter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewConverter(t *testing.T) {
	converter := NewConverter()

	if converter.MaxConcurrency != 1 {
		t.Errorf("Expected MaxConcurrency to be 1, got %d", converter.MaxConcurrency)
	}

	if converter.DeleteSource != false {
		t.Errorf("Expected DeleteSource to be false, got %v", converter.DeleteSource)
	}

	if converter.OutputDir != "." {
		t.Errorf("Expected OutputDir to be '.', got %s", converter.OutputDir)
	}
}

func TestGenerateOutputPath(t *testing.T) {
	converter := NewConverter()

	tests := []struct {
		name      string
		inputFile string
		extension string
		outputDir string
		expected  string
	}{
		{
			name:      "basic conversion",
			inputFile: "test.cbz",
			extension: ".azw3",
			outputDir: ".",
			expected:  "test.azw3",
		},
		{
			name:      "with subdirectory",
			inputFile: "manga/chapter1.cbz",
			extension: ".azw3",
			outputDir: "output",
			expected:  "output/chapter1.azw3",
		},
		{
			name:      "different extension",
			inputFile: "book.cbz",
			extension: ".mobi",
			outputDir: ".",
			expected:  "book.mobi",
		},
		{
			name:      "complex filename",
			inputFile: "/path/to/My Manga - Chapter 01.cbz",
			extension: ".epub",
			outputDir: "converted",
			expected:  "converted/My Manga - Chapter 01.epub",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter.OutputDir = tt.outputDir
			result := converter.GenerateOutputPath(tt.inputFile, tt.extension)
			if result != tt.expected {
				t.Errorf("generateOutputPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		name      string
		format    string
		expectErr bool
	}{
		{
			name:      "valid azw3",
			format:    ".azw3",
			expectErr: false,
		},
		{
			name:      "valid azw3 without dot",
			format:    "azw3",
			expectErr: false,
		},
		{
			name:      "valid mobi",
			format:    ".mobi",
			expectErr: false,
		},
		{
			name:      "valid epub",
			format:    ".epub",
			expectErr: false,
		},
		{
			name:      "valid pdf",
			format:    ".pdf",
			expectErr: false,
		},
		{
			name:      "invalid format",
			format:    ".txt",
			expectErr: true,
		},
		{
			name:      "invalid format without dot",
			format:    "doc",
			expectErr: true,
		},
		{
			name:      "uppercase format",
			format:    ".AZW3",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFormat(tt.format)
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestGetSupportedFormats(t *testing.T) {
	formats := GetSupportedFormats()
	expectedFormats := []string{".azw3", ".mobi", ".epub", ".pdf"}

	if len(formats) != len(expectedFormats) {
		t.Errorf("Expected %d formats, got %d", len(expectedFormats), len(formats))
	}

	for _, expected := range expectedFormats {
		found := false
		for _, format := range formats {
			if format == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected format %s not found in supported formats", expected)
		}
	}
}

func TestIsEbookConvertAvailable(t *testing.T) {
	// This test will pass or fail based on whether Calibre is installed
	// We can't guarantee it's installed in CI, so we'll just check the function works
	available := IsEbookConvertAvailable()

	// The function should return a boolean without error
	if available {
		t.Log("ebook-convert is available on this system")
	} else {
		t.Log("ebook-convert is not available on this system")
	}
}

func TestConvertCBZToAZW3_NonExistentFile(t *testing.T) {
	converter := NewConverter()
	tempDir := t.TempDir()

	inputFile := filepath.Join(tempDir, "nonexistent.cbz")
	outputFile := filepath.Join(tempDir, "output.azw3")

	result, err := converter.ConvertCBZToAZW3(inputFile, outputFile)

	if err == nil {
		t.Error("Expected error for non-existent input file, but got none")
	}

	if result.Success {
		t.Error("Expected conversion to fail for non-existent file")
	}

	if !strings.Contains(result.Error.Error(), "does not exist") {
		t.Errorf("Expected error about file not existing, got: %v", result.Error)
	}
}

func TestConvertCBZToAZW3_OutputDirectoryCreation(t *testing.T) {
	if !IsEbookConvertAvailable() {
		t.Skip("ebook-convert not available, skipping integration test")
	}

	converter := NewConverter()
	tempDir := t.TempDir()

	// Create a dummy CBZ file (actually just an empty file for this test)
	inputFile := filepath.Join(tempDir, "test.cbz")
	if err := os.WriteFile(inputFile, []byte("dummy content"), 0644); err != nil {
		t.Fatalf("Failed to create test input file: %v", err)
	}

	// Output to a nested directory that doesn't exist
	outputFile := filepath.Join(tempDir, "nested", "dir", "output.azw3")

	result, _ := converter.ConvertCBZToAZW3(inputFile, outputFile)

	// Check that the directory was created (even if conversion fails due to invalid CBZ)
	outputDir := filepath.Dir(outputFile)
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Error("Expected output directory to be created")
	}

	// We expect this to fail because our dummy file isn't a valid CBZ
	if result.Success {
		t.Error("Expected conversion to fail for invalid CBZ file")
	}
}

func TestConvertMultiple_EmptyInput(t *testing.T) {
	converter := NewConverter()

	results, err := converter.ConvertMultiple([]string{}, nil)

	if err == nil {
		t.Error("Expected error for empty input files, but got none")
	}

	if results != nil {
		t.Error("Expected nil results for empty input")
	}
}

func TestConvertMultiple_EbookConvertNotAvailable(t *testing.T) {
	if IsEbookConvertAvailable() {
		t.Skip("ebook-convert is available, skipping this test")
	}

	converter := NewConverter()
	inputFiles := []string{"test1.cbz", "test2.cbz"}

	results, err := converter.ConvertMultiple(inputFiles, nil)

	if err == nil {
		t.Error("Expected error when ebook-convert is not available")
	}

	if results != nil {
		t.Error("Expected nil results when ebook-convert is not available")
	}

	if !strings.Contains(err.Error(), "ebook-convert not found") {
		t.Errorf("Expected error about ebook-convert not found, got: %v", err)
	}
}

func TestConvertCBZToMultipleFormats_NoFormats(t *testing.T) {
	converter := NewConverter()
	tempDir := t.TempDir()

	inputFile := filepath.Join(tempDir, "test.cbz")

	results, err := converter.ConvertCBZToMultipleFormats(inputFile, []string{}, nil)

	if err == nil {
		t.Error("Expected error for no output formats, but got none")
	}

	if results != nil {
		t.Error("Expected nil results for no formats")
	}
}

func TestConvertCBZToMultipleFormats_UnsupportedFormat(t *testing.T) {
	if !IsEbookConvertAvailable() {
		t.Skip("ebook-convert not available, skipping integration test")
	}

	converter := NewConverter()
	tempDir := t.TempDir()

	// Create a dummy CBZ file
	inputFile := filepath.Join(tempDir, "test.cbz")
	if err := os.WriteFile(inputFile, []byte("dummy content"), 0644); err != nil {
		t.Fatalf("Failed to create test input file: %v", err)
	}

	formats := []string{".txt", ".doc"} // Unsupported formats

	results, err := converter.ConvertCBZToMultipleFormats(inputFile, formats, nil)

	// Should not error at the function level, but individual results should show errors
	if err != nil {
		t.Errorf("Unexpected function-level error: %v", err)
	}

	if len(results) != len(formats) {
		t.Errorf("Expected %d results, got %d", len(formats), len(results))
	}

	for i, result := range results {
		if result.Success {
			t.Errorf("Expected result %d to fail for unsupported format", i)
		}
		if result.Error == nil {
			t.Errorf("Expected error in result %d for unsupported format", i)
		}
	}
}

func TestConvertCBZToMultipleFormats_FormatNormalization(t *testing.T) {
	if !IsEbookConvertAvailable() {
		t.Skip("ebook-convert not available, skipping integration test")
	}

	converter := NewConverter()
	tempDir := t.TempDir()

	// Create a dummy CBZ file
	inputFile := filepath.Join(tempDir, "test.cbz")
	if err := os.WriteFile(inputFile, []byte("dummy content"), 0644); err != nil {
		t.Fatalf("Failed to create test input file: %v", err)
	}

	// Test format normalization (with and without dots)
	formats := []string{"azw3", ".mobi", "EPUB", ".PDF"}

	results, err := converter.ConvertCBZToMultipleFormats(inputFile, formats, nil)

	if err != nil {
		t.Errorf("Unexpected function-level error: %v", err)
	}

	if len(results) != len(formats) {
		t.Errorf("Expected %d results, got %d", len(formats), len(results))
	}

	// Check that output files have correct extensions
	expectedExtensions := []string{".azw3", ".mobi", ".epub", ".pdf"}
	for i, result := range results {
		expectedExt := expectedExtensions[i]
		if !strings.HasSuffix(result.OutputFile, expectedExt) {
			t.Errorf("Expected result %d to have extension %s, got %s", i, expectedExt, result.OutputFile)
		}
	}
}

func TestConverterConcurrency(t *testing.T) {
	converter := NewConverter()
	converter.MaxConcurrency = 2

	if converter.MaxConcurrency != 2 {
		t.Errorf("Expected MaxConcurrency to be 2, got %d", converter.MaxConcurrency)
	}
}

func TestConverterDeleteSource(t *testing.T) {
	converter := NewConverter()
	converter.DeleteSource = true

	if !converter.DeleteSource {
		t.Error("Expected DeleteSource to be true")
	}
}

func TestConverterOutputDir(t *testing.T) {
	converter := NewConverter()
	converter.OutputDir = "/custom/output"

	if converter.OutputDir != "/custom/output" {
		t.Errorf("Expected OutputDir to be '/custom/output', got %s", converter.OutputDir)
	}
}

func TestConversionResultStruct(t *testing.T) {
	result := &ConversionResult{
		InputFile:    "input.cbz",
		OutputFile:   "output.azw3",
		Success:      true,
		Error:        nil,
		BytesWritten: 1024,
	}

	if result.InputFile != "input.cbz" {
		t.Errorf("Expected InputFile to be 'input.cbz', got %s", result.InputFile)
	}

	if result.OutputFile != "output.azw3" {
		t.Errorf("Expected OutputFile to be 'output.azw3', got %s", result.OutputFile)
	}

	if !result.Success {
		t.Error("Expected Success to be true")
	}

	if result.Error != nil {
		t.Errorf("Expected Error to be nil, got %v", result.Error)
	}

	if result.BytesWritten != 1024 {
		t.Errorf("Expected BytesWritten to be 1024, got %d", result.BytesWritten)
	}
}
