package converter

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// ConversionResult represents the result of a conversion operation
type ConversionResult struct {
	InputFile    string
	OutputFile   string
	Success      bool
	Error        error
	BytesWritten int64
}

// ProgressCallback is called during conversion progress
type ProgressCallback func(current, total int, result *ConversionResult)

// Converter handles file format conversions using external tools
type Converter struct {
	// MaxConcurrency limits the number of concurrent conversions
	MaxConcurrency int
	// DeleteSource determines whether to delete source files after successful conversion
	DeleteSource bool
	// OutputDir is the directory where converted files will be saved
	OutputDir string
}

// NewConverter creates a new converter with default settings
func NewConverter() *Converter {
	return &Converter{
		MaxConcurrency: 1, // Conservative default to avoid overwhelming the system
		DeleteSource:   true,
		OutputDir:      ".",
	}
}

// ConvertCBZToAZW3 converts a CBZ file to AZW3 format using Calibre's ebook-convert
func (c *Converter) ConvertCBZToAZW3(inputFile string, outputFile string) (*ConversionResult, error) {
	result := &ConversionResult{
		InputFile:  inputFile,
		OutputFile: outputFile,
	}

	// Check if input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		result.Error = fmt.Errorf("input file does not exist: %s", inputFile)
		return result, result.Error
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create output directory: %w", err)
		return result, result.Error
	}

	// Check if ebook-convert is available
	if err := c.checkEbookConvert(); err != nil {
		result.Error = err
		return result, result.Error
	}

	// Run ebook-convert command
	cmd := exec.Command("/Applications/calibre.app/Contents/MacOS/ebook-convert", inputFile, outputFile)

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Error = fmt.Errorf("ebook-convert failed: %w\nOutput: %s", err, string(output))
		return result, result.Error
	}

	// Check if output file was created
	if stat, err := os.Stat(outputFile); err != nil {
		result.Error = fmt.Errorf("output file was not created: %s", outputFile)
		return result, result.Error
	} else {
		result.BytesWritten = stat.Size()
	}

	result.Success = true

	if err := os.Remove(inputFile); err != nil {
		// Don't fail the conversion if we can't delete the source
		result.Error = fmt.Errorf("conversion successful but failed to delete source file: %w", err)
	}

	return result, nil
}

// ConvertCBZToEPUB converts a CBZ file to EPUB format using Calibre's ebook-convert
func (c *Converter) ConvertCBZToEPUB(inputFile string, outputFile string) (*ConversionResult, error) {
	result := &ConversionResult{
		InputFile:  inputFile,
		OutputFile: outputFile,
	}

	// Check if input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		result.Error = fmt.Errorf("input file does not exist: %s", inputFile)
		return result, result.Error
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create output directory: %w", err)
		return result, result.Error
	}

	// Check if ebook-convert is available
	if err := c.checkEbookConvert(); err != nil {
		result.Error = err
		return result, result.Error
	}

	// Run ebook-convert command
	cmd := exec.Command("/Applications/calibre.app/Contents/MacOS/ebook-convert", inputFile, outputFile)

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Error = fmt.Errorf("ebook-convert failed: %w\nOutput: %s", err, string(output))
		return result, result.Error
	}

	// Check if output file was created
	if stat, err := os.Stat(outputFile); err != nil {
		result.Error = fmt.Errorf("output file was not created: %s", outputFile)
		return result, result.Error
	} else {
		result.BytesWritten = stat.Size()
	}

	result.Success = true

	if err := os.Remove(inputFile); err != nil {
		// Don't fail the conversion if we can't delete the source
		result.Error = fmt.Errorf("conversion successful but failed to delete source file: %w", err)
	}

	return result, nil
}

// ConvertMultiple converts multiple CBZ files to AZW3 format concurrently
func (c *Converter) ConvertMultiple(inputFiles []string, progress ProgressCallback) ([]*ConversionResult, error) {
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("no input files provided")
	}

	// Check if ebook-convert is available before starting
	if err := c.checkEbookConvert(); err != nil {
		return nil, err
	}

	results := make([]*ConversionResult, len(inputFiles))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, c.MaxConcurrency)

	for i, inputFile := range inputFiles {
		wg.Add(1)
		go func(index int, input string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Generate output filename
			outputFile := c.GenerateOutputPath(input, ".azw3")

			// Perform conversion
			result, _ := c.ConvertCBZToAZW3(input, outputFile)
			results[index] = result

			// Report progress
			if progress != nil {
				progress(index+1, len(inputFiles), result)
			}
		}(i, inputFile)
	}

	wg.Wait()
	return results, nil
}

// ConvertCBZToMultipleFormats converts a CBZ file to multiple output formats
func (c *Converter) ConvertCBZToMultipleFormats(inputFile string, formats []string, progress ProgressCallback) ([]*ConversionResult, error) {
	if len(formats) == 0 {
		return nil, fmt.Errorf("no output formats specified")
	}

	// Check if ebook-convert is available
	if err := c.checkEbookConvert(); err != nil {
		return nil, err
	}

	results := make([]*ConversionResult, len(formats))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, c.MaxConcurrency)

	for i, format := range formats {
		wg.Add(1)
		go func(index int, format string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Ensure format starts with dot
			if !strings.HasPrefix(format, ".") {
				format = "." + format
			}

			// Generate output filename
			outputFile := c.GenerateOutputPath(inputFile, format)

			// Perform conversion based on format
			var result *ConversionResult
			var err error

			switch strings.ToLower(format) {
			case ".azw3":
				result, err = c.ConvertCBZToAZW3(inputFile, outputFile)
			case ".epub":
				result, err = c.ConvertCBZToFormat(inputFile, outputFile, "epub")
			case ".mobi":
				result, err = c.ConvertCBZToFormat(inputFile, outputFile, "mobi")
			case ".pdf":
				result, err = c.ConvertCBZToFormat(inputFile, outputFile, "pdf")
			default:
				result = &ConversionResult{
					InputFile:  inputFile,
					OutputFile: outputFile,
					Success:    false,
					Error:      fmt.Errorf("unsupported output format: %s", format),
				}
			}

			if err != nil && result.Error == nil {
				result.Error = err
			}

			results[index] = result

			// Report progress
			if progress != nil {
				progress(index+1, len(formats), result)
			}
		}(i, format)
	}

	wg.Wait()
	return results, nil
}

// GenerateOutputPath generates the output file path based on input file and extension
func (c *Converter) GenerateOutputPath(inputFile, extension string) string {
	baseName := strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))
	outputFile := baseName + extension

	if c.OutputDir != "" && c.OutputDir != "." {
		outputFile = filepath.Join(c.OutputDir, outputFile)
	}

	return outputFile
}

// ConvertCBZToFormat is a generic conversion function for any format supported by ebook-convert
func (c *Converter) ConvertCBZToFormat(inputFile, outputFile, format string) (*ConversionResult, error) {
	result := &ConversionResult{
		InputFile:  inputFile,
		OutputFile: outputFile,
	}

	// Check if input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		result.Error = fmt.Errorf("input file does not exist: %s", inputFile)
		return result, result.Error
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create output directory: %w", err)
		return result, result.Error
	}

	// Run ebook-convert command
	cmd := exec.Command("ebook-convert", inputFile, outputFile)

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Error = fmt.Errorf("ebook-convert to %s failed: %w\nOutput: %s", format, err, string(output))
		return result, result.Error
	}

	// Check if output file was created
	if stat, err := os.Stat(outputFile); err != nil {
		result.Error = fmt.Errorf("output file was not created: %s", outputFile)
		return result, result.Error
	} else {
		result.BytesWritten = stat.Size()
	}

	result.Success = true

	if err := os.Remove(inputFile); err != nil {
		// Don't fail the conversion if we can't delete the source
		result.Error = fmt.Errorf("conversion successful but failed to delete source file: %w", err)
	}

	return result, nil
}

// checkEbookConvert verifies that ebook-convert is available
func (c *Converter) checkEbookConvert() error {
	_, err := exec.LookPath("/Applications/calibre.app/Contents/MacOS/ebook-convert")
	if err != nil {
		return fmt.Errorf("ebook-convert not found. Please install Calibre: https://calibre-ebook.com/download")
	}
	return nil
}

// IsEbookConvertAvailable checks if ebook-convert is available on the system
func IsEbookConvertAvailable() bool {
	_, err := exec.LookPath("/Applications/calibre.app/Contents/MacOS/ebook-convert")
	return err == nil
}

// GetSupportedFormats returns a list of formats supported for conversion
func GetSupportedFormats() []string {
	return []string{".azw3", ".mobi", ".epub", ".pdf"}
}

// ValidateFormat checks if the given format is supported
func ValidateFormat(format string) error {
	if !strings.HasPrefix(format, ".") {
		format = "." + format
	}

	supportedFormats := GetSupportedFormats()
	for _, supported := range supportedFormats {
		if strings.ToLower(format) == supported {
			return nil
		}
	}

	return fmt.Errorf("unsupported format: %s. Supported formats: %v", format, supportedFormats)
}
