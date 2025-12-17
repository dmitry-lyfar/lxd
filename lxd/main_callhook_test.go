package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveTargetRelativeToLink(t *testing.T) {
	tests := []struct {
		name      string
		link      string
		target    string
		expected  string
		expectErr bool
	}{
		{
			name:      "relative target returned as-is",
			link:      "/home/user/link",
			target:    "relative/path",
			expected:  "relative/path",
			expectErr: false,
		},
		{
			name:      "absolute target in same directory",
			link:      "/home/user/link",
			target:    "/home/user/target",
			expected:  "target",
			expectErr: false,
		},
		{
			name:      "absolute target in subdirectory",
			link:      "/home/user/link",
			target:    "/home/user/subdir/target",
			expected:  filepath.Join("subdir", "target"),
			expectErr: false,
		},
		{
			name:      "absolute target in parent directory",
			link:      "/home/user/subdir/link",
			target:    "/home/user/target",
			expected:  filepath.Join("..", "target"),
			expectErr: false,
		},
		{
			name:      "absolute target in sibling directory",
			link:      "/home/user/dir1/link",
			target:    "/home/user/dir2/target",
			expected:  filepath.Join("..", "dir2", "target"),
			expectErr: false,
		},
		{
			name:      "absolute target at root level",
			link:      "/home/user/link",
			target:    "/target",
			expected:  filepath.Join("..", "..", "target"),
			expectErr: false,
		},
		{
			name:      "paths with trailing slashes get cleaned",
			link:      "/home/user/link",
			target:    "/home/user/target/",
			expected:  "target",
			expectErr: false,
		},
		{
			name:      "paths with redundant separators get cleaned",
			link:      "/home//user///link",
			target:    "/home//user//target",
			expected:  "target",
			expectErr: false,
		},
		{
			name:      "paths with dot components get cleaned",
			link:      "/home/./user/link",
			target:    "/home/user/./target",
			expected:  "target",
			expectErr: false,
		},
		{
			name:      "relative link path returns error",
			link:      "relative/link",
			target:    "/absolute/target",
			expected:  "",
			expectErr: true,
		},
		{
			name:      "empty link path returns error",
			link:      "",
			target:    "/absolute/target",
			expected:  "",
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := resolveTargetRelativeToLink(tc.link, tc.target)

			if tc.expectErr {
				assert.Error(t, err, "Expected an error for link=%q and target=%q", tc.link, tc.target)
			} else {
				assert.NoError(t, err, "Expected no error for link=%q and target=%q", tc.link, tc.target)
			}

			assert.Equal(t, tc.expected, result, "Unexpected result value %q for link=%q and target=%q", result, tc.link, tc.target)
		})
	}
}

func TestApplyCDIHooksCreatesLDConf(t *testing.T) {
	// Prepare temporary devices root and container rootfs
	devicesRoot := t.TempDir()
	containerRoot := t.TempDir()

	// Ensure env vars are set for the function
	t.Setenv("LXC_ROOTFS_MOUNT", containerRoot)
	// Skip running ldconfig in tests
	t.Setenv("LXD_SKIP_LDCONFIG", "1")

	// Create a hooks file with a single LD cache update
	hooksFileName := "test_cdi_hooks.json"
	hooksFilePath := filepath.Join(devicesRoot, hooksFileName)
	content := `{"ld_cache_updates":["/opt/test/lib"]}`
	if err := os.WriteFile(hooksFilePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write hooks file: %v", err)
	}

	// Run the function under test
	if err := applyCDIHooksToContainer(devicesRoot, hooksFileName); err != nil {
		t.Fatalf("applyCDIHooksToContainer failed: %v", err)
	}

	// Check that the ld.so.conf.d directory and file were created and contain the entry
	ldConfDir := filepath.Join(containerRoot, "etc", "ld.so.conf.d")
	ldConfFile := filepath.Join(ldConfDir, customCDILinkerConfFile)

	data, err := os.ReadFile(ldConfFile)
	if err != nil {
		t.Fatalf("failed to read ld conf file: %v", err)
	}

	if !strings.Contains(string(data), "/opt/test/lib") {
		t.Fatalf("ld conf file does not contain expected entry; got: %q", string(data))
	}
}
