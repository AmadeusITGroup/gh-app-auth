package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath_HomeDirectory(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "tilde only",
			path:    "~",
			wantErr: false,
		},
		{
			name:    "tilde with subdirectory",
			path:    "~/.config/gh-app-auth",
			wantErr: false,
		},
		{
			name:    "tilde with file",
			path:    "~/key.pem",
			wantErr: false,
		},
		{
			name:    "absolute path",
			path:    "/tmp/key.pem",
			wantErr: false,
		},
		{
			name:    "relative path",
			path:    "./key.pem",
			wantErr: false,
		},
		{
			name:    "current directory",
			path:    ".",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expanded, err := expandPath(tt.path)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify expansion happened
			if expanded == "" {
				t.Error("Expanded path should not be empty")
			}

			// For tilde paths, verify they were expanded
			if tt.path[0] == '~' && expanded[0] == '~' {
				t.Error("Tilde should have been expanded")
			}

			// For absolute paths starting with /, verify they're absolute
			if filepath.IsAbs(tt.path) && !filepath.IsAbs(expanded) {
				t.Errorf("Absolute path %q became relative: %q", tt.path, expanded)
			}
		})
	}
}

func TestExpandHomeDirectory_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "tilde only",
			path:    "~",
			wantErr: false,
		},
		{
			name:    "tilde with slash",
			path:    "~/",
			wantErr: false,
		},
		{
			name:    "tilde with subpath",
			path:    "~/subdir/file",
			wantErr: false,
		},
		{
			name:    "tilde with hidden dir",
			path:    "~/.hidden",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandHomeDirectory(tt.path)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify result doesn't start with tilde
			if result[0] == '~' {
				t.Errorf("Result still has tilde: %q", result)
			}

			// Verify result is absolute
			if !filepath.IsAbs(result) {
				t.Errorf("Result should be absolute: %q", result)
			}

			// For tilde only, should equal home directory
			if tt.path == "~" {
				homeDir, _ := os.UserHomeDir()
				if result != homeDir {
					t.Errorf("Expected home dir %q, got %q", homeDir, result)
				}
			}
		})
	}
}

func TestExpandPath_Consistency(t *testing.T) {
	// Same path should always expand to the same result
	path := "~/test.txt"

	result1, err1 := expandPath(path)
	result2, err2 := expandPath(path)

	if err1 != nil || err2 != nil {
		t.Fatalf("Unexpected errors: %v, %v", err1, err2)
	}

	if result1 != result2 {
		t.Errorf("Inconsistent expansion: %q vs %q", result1, result2)
	}
}

func TestExpandPath_AbsolutePaths(t *testing.T) {
	tempDir := t.TempDir()
	testPath := filepath.Join(tempDir, "test.txt")

	expanded, err := expandPath(testPath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if expanded != testPath {
		t.Errorf("Absolute path changed: got %q, want %q", expanded, testPath)
	}
}

func TestGetDefaultConfigPath_ReturnsPath(t *testing.T) {
	path := getDefaultConfigPath()

	if path == "" {
		t.Error("Expected non-empty path")
	}

	if !filepath.IsAbs(path) {
		t.Errorf("Expected absolute path, got %q", path)
	}

	// Should contain gh-app-auth in the path
	if !contains(path, "gh-app-auth") {
		t.Errorf("Path should contain 'gh-app-auth': %q", path)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || filepath.Base(filepath.Dir(s)) == substr || filepath.Base(s) == substr+".yml")
}
