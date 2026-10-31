package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCheckFromWorkingDirectory(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := run([]string{"--check"}, &stdout, &stderr)
	if status != 0 {
		t.Fatalf("run --check status = %d, want 0 (stderr: %s)", status, stderr.String())
	}
	if !strings.Contains(stdout.String(), "up to date") {
		t.Fatalf("run --check output = %q, want up to date", stdout.String())
	}
}

func TestRunRejectsUnknownFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if status := run([]string{"--unknown"}, &stdout, &stderr); status != 2 {
		t.Fatalf("run unknown flag status = %d, want 2", status)
	}
	if !strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Fatalf("run unknown flag stderr = %q, want flag error", stderr.String())
	}
}

func TestRunInRejectsUnknownFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if status := runIn(t.TempDir(), []string{"--unknown"}, &stdout, &stderr); status != 2 {
		t.Fatalf("runIn unknown flag status = %d, want 2", status)
	}
}

func TestRunInReportsMissingRepositoryRoot(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := runIn(t.TempDir(), nil, &stdout, &stderr)
	if status != 1 {
		t.Fatalf("runIn status = %d, want 1", status)
	}
	if !strings.Contains(stderr.String(), "go.mod not found above") {
		t.Fatalf("runIn stderr = %q, want missing module error", stderr.String())
	}
}

func TestRunInReportsGenerationFailure(t *testing.T) {
	root := tempRepo(t)
	unsupported := `{"$defs": {"TopLevelArray": {"type": "array", "items": {"type": "string"}}}}`
	if err := os.WriteFile(filepath.Join(root, "protocol", "schemas", "unsupported.schema.json"), []byte(unsupported), 0o644); err != nil {
		t.Fatalf("write unsupported schema: %v", err)
	}

	var stdout, stderr bytes.Buffer
	if status := runIn(root, nil, &stdout, &stderr); status != 1 {
		t.Fatalf("runIn status = %d, want 1", status)
	}
	if !strings.Contains(stderr.String(), "unsupported definition kind") {
		t.Fatalf("runIn stderr = %q, want generation failure", stderr.String())
	}
}

func TestRunInReportsUnreadableArtifact(t *testing.T) {
	root := tempRepo(t)
	var stdout, stderr bytes.Buffer
	if status := runIn(root, nil, &stdout, &stderr); status != 0 {
		t.Fatalf("generate status = %d, want 0 (stderr: %s)", status, stderr.String())
	}

	target := filepath.Join(root, filepath.FromSlash(typesGoPath))
	if err := os.Remove(target); err != nil {
		t.Fatalf("remove generated file: %v", err)
	}
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("replace generated file with directory: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if status := runIn(root, []string{"--check"}, &stdout, &stderr); status != 1 {
		t.Fatalf("check status = %d, want 1", status)
	}
	if !strings.Contains(stderr.String(), "read "+typesGoPath) {
		t.Fatalf("check stderr = %q, want read failure", stderr.String())
	}
}

func TestFindRepoRootRejectsTreeWithoutModule(t *testing.T) {
	if _, err := findRepoRoot(t.TempDir()); err == nil || !strings.Contains(err.Error(), "go.mod not found above") {
		t.Fatalf("findRepoRoot error = %v, want missing module", err)
	}
}

func TestFindRepoRootAcceptsRelativePath(t *testing.T) {
	root := repoRoot(t)
	found, err := findRepoRoot(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}
	if found != root {
		t.Fatalf("findRepoRoot = %q, want %q", found, root)
	}
}

func TestWriteArtifactsErrors(t *testing.T) {
	t.Run("directory creation", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "internal"), []byte("file"), 0o644); err != nil {
			t.Fatalf("write blocking file: %v", err)
		}
		err := writeArtifacts(root, []artifact{{Path: "internal/protocol/types.generated.go", Content: []byte("x")}})
		if err == nil || !strings.Contains(err.Error(), "create directory for") {
			t.Fatalf("writeArtifacts error = %v, want directory failure", err)
		}
	})
	t.Run("file write", func(t *testing.T) {
		root := t.TempDir()
		target := filepath.Join(root, "internal", "protocol", "types.generated.go")
		if err := os.MkdirAll(target, 0o755); err != nil {
			t.Fatalf("create blocking directory: %v", err)
		}
		err := writeArtifacts(root, []artifact{{Path: "internal/protocol/types.generated.go", Content: []byte("x")}})
		if err == nil || !strings.Contains(err.Error(), "write internal/protocol/types.generated.go") {
			t.Fatalf("writeArtifacts error = %v, want write failure", err)
		}
	})
}

func TestWriteArtifactsWritesContent(t *testing.T) {
	root := t.TempDir()
	artifacts := []artifact{{Path: "nested/generated.txt", Content: []byte("content")}}
	if err := writeArtifacts(root, artifacts); err != nil {
		t.Fatalf("writeArtifacts: %v", err)
	}
	written, err := os.ReadFile(filepath.Join(root, "nested", "generated.txt"))
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	if string(written) != "content" {
		t.Fatalf("artifact = %q, want content", written)
	}
}

func TestGenerateArtifactsProducesEveryArtifact(t *testing.T) {
	artifacts, err := generateArtifacts(repoRoot(t))
	if err != nil {
		t.Fatalf("generateArtifacts: %v", err)
	}
	if len(artifacts) != 4 {
		t.Fatalf("artifacts = %d, want 4", len(artifacts))
	}
	for _, item := range artifacts {
		if len(item.Content) == 0 {
			t.Errorf("artifact %s is empty", item.Path)
		}
	}
}
