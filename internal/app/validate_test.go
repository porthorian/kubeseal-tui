package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAndPrepareDefaultsTypeAndComputesFiles(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		SecretName:          " app-secret ",
		ControllerNamespace: " sealed-secrets ",
		ControllerName:      " sealed-secrets ",
		CWD:                 dir,
		Data: []DataEntry{
			{Key: "token", Source: DataSourceValue, Payload: "value"},
		},
		Targets: []Target{
			{Namespace: " prod ", Dir: dir},
		},
	}

	if err := ValidateAndPrepare(&cfg); err != nil {
		t.Fatalf("ValidateAndPrepare returned error: %v", err)
	}

	if cfg.SecretType != "Opaque" {
		t.Fatalf("expected default secret type Opaque, got %q", cfg.SecretType)
	}
	if cfg.Targets[0].Namespace != "prod" {
		t.Fatalf("expected trimmed namespace, got %q", cfg.Targets[0].Namespace)
	}
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks returned error: %v", err)
	}
	if cfg.Targets[0].File != filepath.Join(resolvedDir, "sealed-app-secret.yaml") {
		t.Fatalf("unexpected target file: %q", cfg.Targets[0].File)
	}
}

func TestValidateAndPrepareRejectsDuplicateOutputFiles(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		SecretName:          "app-secret",
		SecretType:          "Opaque",
		ControllerNamespace: "sealed-secrets",
		ControllerName:      "sealed-secrets",
		CWD:                 dir,
		Data: []DataEntry{
			{Key: "token", Source: DataSourceValue, Payload: "value"},
		},
		Targets: []Target{
			{Namespace: "prod", Dir: dir},
			{Namespace: "stage", Dir: dir},
		},
	}

	err := ValidateAndPrepare(&cfg)
	if err == nil {
		t.Fatal("expected duplicate output error")
	}
	if !strings.Contains(err.Error(), "duplicate output file path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExistingOutputFiles(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "sealed-app-secret.yaml")
	if err := os.WriteFile(existing, []byte("existing"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	got := ExistingOutputFiles([]Target{
		{File: existing},
		{File: filepath.Join(dir, "missing.yaml")},
	})

	if len(got) != 1 || got[0] != existing {
		t.Fatalf("unexpected existing files: %#v", got)
	}
}
