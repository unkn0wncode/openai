package openai

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExamples(t *testing.T) {
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		t.Skip("OPENAI_API_KEY not set, skip testing examples")
	}

	exampleDirs, err := collectExampleDirs(t, "examples")
	require.NoError(t, err)
	require.NotEmpty(t, exampleDirs)

	for _, dir := range exampleDirs {
		name := strings.ReplaceAll(dir, string(filepath.Separator), "_")
		t.Run(name, func(t *testing.T) {
			cmd := exec.Command("go", "run", ".")
			cmd.Dir = filepath.FromSlash(dir)
			cmd.Env = append(os.Environ(), "OPENAI_API_KEY="+token)
			output, err := cmd.CombinedOutput()
			require.NoError(t, err)
			t.Logf("example %s output:\n%s", dir, string(output))
		})
	}
}

// collectExampleDirs walks root recursively and returns directories that contain a main.go file.
func collectExampleDirs(t *testing.T, root string) ([]string, error) {
	var dirs []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		require.NoError(t, err)
		if !d.IsDir() {
			return nil
		}
		if _, statErr := os.Stat(filepath.Join(path, "main.go")); statErr == nil {
			dirs = append(dirs, path)
			t.Logf("found example directory: %s", path)
		}
		return nil
	})
	return dirs, err
}
