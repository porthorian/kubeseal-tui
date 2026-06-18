package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDataFlagValuePreservesValueAfterFirstEquals(t *testing.T) {
	var cfg Config

	if err := ParseDataFlag(&cfg, " token =abc=123", DataSourceValue); err != nil {
		t.Fatalf("ParseDataFlag returned error: %v", err)
	}

	if len(cfg.Data) != 1 {
		t.Fatalf("expected 1 data entry, got %d", len(cfg.Data))
	}
	if cfg.Data[0].Key != "token" {
		t.Fatalf("expected trimmed key, got %q", cfg.Data[0].Key)
	}
	if cfg.Data[0].Payload != "abc=123" {
		t.Fatalf("expected value after first equals to be preserved, got %q", cfg.Data[0].Payload)
	}
}

func TestAddDataRejectsDuplicateKeys(t *testing.T) {
	var cfg Config

	if err := AddDataValue(&cfg, "token", "one"); err != nil {
		t.Fatalf("AddDataValue returned error: %v", err)
	}
	if err := AddDataValue(&cfg, "token", "two"); err == nil {
		t.Fatal("expected duplicate key error")
	}
}

func TestAddTargetResolvesRelativeDirFromCWD(t *testing.T) {
	cwd := t.TempDir()
	targetDir := filepath.Join(cwd, "manifests")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatalf("Mkdir returned error: %v", err)
	}
	cfg := Config{CWD: cwd}

	if err := AddTarget(&cfg, "default", "manifests"); err != nil {
		t.Fatalf("AddTarget returned error: %v", err)
	}

	want, err := filepath.EvalSymlinks(targetDir)
	if err != nil {
		t.Fatalf("EvalSymlinks returned error: %v", err)
	}
	if cfg.Targets[0].Dir != want {
		t.Fatalf("expected target dir %q, got %q", want, cfg.Targets[0].Dir)
	}
}
