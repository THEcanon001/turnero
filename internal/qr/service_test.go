package qr_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/THEcanon001/turnero/internal/qr"
)

func TestGenerateBytes(t *testing.T) {
	s := qr.NewService("https://turnero.app", "/tmp/qr-test")

	png, err := s.GenerateBytes("test-slug")
	require.NoError(t, err)
	assert.NotEmpty(t, png)
	// PNG magic bytes
	assert.Equal(t, byte(0x89), png[0])
	assert.Equal(t, byte('P'), png[1])
	assert.Equal(t, byte('N'), png[2])
	assert.Equal(t, byte('G'), png[3])
}

func TestGenerate(t *testing.T) {
	tmpDir := t.TempDir()
	s := qr.NewService("https://turnero.app", tmpDir)

	filePath, err := s.Generate("test-provider")
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(tmpDir, "test-provider.png"), filePath)

	// Verify file exists and is a valid PNG
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.True(t, len(data) > 100)
	assert.Equal(t, byte(0x89), data[0])
}
