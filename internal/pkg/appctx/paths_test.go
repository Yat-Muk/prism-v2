package appctx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPaths(t *testing.T) {
	tmpDir := t.TempDir()

	paths, err := NewPaths(tmpDir)
	require.NoError(t, err)

	assert.NotNil(t, paths)
	assert.Equal(t, tmpDir, paths.BaseDir)
}

func TestPaths_Directories(t *testing.T) {
	tmpDir := t.TempDir()

	paths, err := NewPaths(tmpDir)
	require.NoError(t, err)

	// 验证目录路径不为空
	assert.NotEmpty(t, paths.CertDir)
	assert.NotEmpty(t, paths.ConfigDir)

	// 验证目录已创建
	assert.DirExists(t, paths.CertDir)
	assert.DirExists(t, paths.ConfigDir)
}
