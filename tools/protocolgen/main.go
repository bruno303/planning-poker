package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	typesGoPath    = "internal/protocol/types.generated.go"
	messagesGoPath = "internal/protocol/messages.generated.go"
)

// artifact is one generated file.
type artifact struct {
	Path    string
	Content []byte
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	workingDirectory, err := os.Getwd()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "protocolgen: %v\n", err)
		return 1
	}
	return runIn(workingDirectory, args, stdout, stderr)
}

// runIn executes the CLI as if the process started in startDirectory.
func runIn(startDirectory string, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("protocolgen", flag.ContinueOnError)
	flags.SetOutput(stderr)
	check := flags.Bool("check", false, "verify the generated files are up to date and exit 1 when they are not")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	root, err := findRepoRoot(startDirectory)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "protocolgen: %v\n", err)
		return 1
	}

	artifacts, err := generateArtifacts(root)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "protocolgen: %v\n", err)
		return 1
	}

	if *check {
		stale, err := staleArtifacts(root, artifacts)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "protocolgen: %v\n", err)
			return 1
		}
		if len(stale) > 0 {
			for _, path := range stale {
				_, _ = fmt.Fprintf(stderr, "stale generated file: %s\n", path)
			}
			return 1
		}
		_, _ = fmt.Fprintln(stdout, "protocol artifacts are up to date")
		return 0
	}

	if err := writeArtifacts(root, artifacts); err != nil {
		_, _ = fmt.Fprintf(stderr, "protocolgen: %v\n", err)
		return 1
	}
	for _, item := range artifacts {
		_, _ = fmt.Fprintf(stdout, "wrote %s\n", item.Path)
	}
	return 0
}

// findRepoRoot walks up from start until it finds the module's go.mod.
func findRepoRoot(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("stat go.mod in %s: %w", current, err)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("go.mod not found above %s", start)
		}
		current = parent
	}
}

func generateArtifacts(root string) ([]artifact, error) {
	spec, err := loadSpec(root)
	if err != nil {
		return nil, err
	}

	types, err := emitGoTypes(spec)
	if err != nil {
		return nil, fmt.Errorf("generate %s: %w", typesGoPath, err)
	}
	messages, err := emitGoMessages(spec)
	if err != nil {
		return nil, fmt.Errorf("generate %s: %w", messagesGoPath, err)
	}
	typescript, err := emitTypeScript(spec)
	if err != nil {
		return nil, fmt.Errorf("generate %s: %w", typescriptOutputPath, err)
	}
	documentation, err := emitDocs(spec)
	if err != nil {
		return nil, fmt.Errorf("generate %s: %w", protocolReadmePath, err)
	}

	return []artifact{
		{Path: typesGoPath, Content: types},
		{Path: messagesGoPath, Content: messages},
		{Path: typescriptOutputPath, Content: typescript},
		{Path: protocolReadmePath, Content: documentation},
	}, nil
}

func writeArtifacts(root string, artifacts []artifact) error {
	for _, item := range artifacts {
		target := filepath.Join(root, filepath.FromSlash(item.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create directory for %s: %w", item.Path, err)
		}
		if err := os.WriteFile(target, item.Content, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", item.Path, err)
		}
	}
	return nil
}

func staleArtifacts(root string, artifacts []artifact) ([]string, error) {
	var stale []string
	for _, item := range artifacts {
		target := filepath.Join(root, filepath.FromSlash(item.Path))
		current, err := os.ReadFile(target)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				stale = append(stale, item.Path)
				continue
			}
			return nil, fmt.Errorf("read %s: %w", item.Path, err)
		}
		if !bytes.Equal(current, item.Content) {
			stale = append(stale, item.Path)
		}
	}
	return stale, nil
}
