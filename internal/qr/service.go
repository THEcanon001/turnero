package qr

import (
	"fmt"
	"os"
	"path/filepath"

	qrcode "github.com/skip2/go-qrcode"
)

// Service handles QR code generation.
type Service struct {
	baseURL   string
	outputDir string
}

// NewService creates a new QR service.
func NewService(baseURL, outputDir string) *Service {
	return &Service{
		baseURL:   baseURL,
		outputDir: outputDir,
	}
}

// Generate creates a QR code PNG for the given provider slug and returns the file path.
func (s *Service) Generate(slug string) (string, error) {
	url := fmt.Sprintf("%s/providers/%s", s.baseURL, slug)

	if err := os.MkdirAll(s.outputDir, 0755); err != nil {
		return "", fmt.Errorf("qr.service: create dir: %w", err)
	}

	filename := slug + ".png"
	filePath := filepath.Join(s.outputDir, filename)

	if err := qrcode.WriteFile(url, qrcode.Medium, 512, filePath); err != nil {
		return "", fmt.Errorf("qr.service: generate: %w", err)
	}

	return filePath, nil
}

// GenerateBytes creates a QR code PNG in memory and returns the bytes.
func (s *Service) GenerateBytes(slug string) ([]byte, error) {
	url := fmt.Sprintf("%s/providers/%s", s.baseURL, slug)

	png, err := qrcode.Encode(url, qrcode.Medium, 512)
	if err != nil {
		return nil, fmt.Errorf("qr.service: encode: %w", err)
	}

	return png, nil
}
