package handlers

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePublishPaths(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "public")
	gotRoot, gotOutput, err := validatePublishPaths(root, output)
	if err != nil {
		t.Fatal(err)
	}
	if gotRoot != root || gotOutput != output {
		t.Fatalf("paths = %q, %q", gotRoot, gotOutput)
	}
	if _, _, err := validatePublishPaths(root, root); err == nil {
		t.Fatal("expected source/output equality to be rejected")
	}
}

func TestLimitPublishOutput(t *testing.T) {
	got := limitPublishOutput(strings.Repeat("x", 20), 8)
	if got != "xxxxxxxx\n...[truncated]" {
		t.Fatalf("unexpected output %q", got)
	}
}
