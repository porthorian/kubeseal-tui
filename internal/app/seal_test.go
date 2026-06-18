package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRunner struct {
	seenManifests []string
}

func (r *fakeRunner) Seal(_ context.Context, opts KubesealOptions, secretYAML []byte) ([]byte, error) {
	r.seenManifests = append(r.seenManifests, string(secretYAML))
	return []byte("sealed by " + opts.ControllerNamespace + "/" + opts.ControllerName + "\n"), nil
}

func TestSealAndWriteAllUsesRunnerAndWritesTarget(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		SecretName:          "app-secret",
		SecretType:          "Opaque",
		ControllerNamespace: "sealed-secrets",
		ControllerName:      "sealed-secrets",
		CWD:                 dir,
		Data: []DataEntry{
			{Key: "token", Source: DataSourceValue, Payload: "plain-secret"},
		},
		Targets: []Target{
			{Namespace: "prod", Dir: dir},
		},
	}
	if err := ValidateAndPrepare(&cfg); err != nil {
		t.Fatalf("ValidateAndPrepare returned error: %v", err)
	}

	runner := &fakeRunner{}
	var out bytes.Buffer
	if err := SealAndWriteAll(context.Background(), cfg, runner, &out); err != nil {
		t.Fatalf("SealAndWriteAll returned error: %v", err)
	}

	written, err := os.ReadFile(filepath.Join(dir, "sealed-app-secret.yaml"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(written) != "sealed by sealed-secrets/sealed-secrets\n" {
		t.Fatalf("unexpected sealed output: %q", string(written))
	}
	if len(runner.seenManifests) != 1 {
		t.Fatalf("expected 1 manifest, got %d", len(runner.seenManifests))
	}
	if !strings.Contains(runner.seenManifests[0], "  'token': 'cGxhaW4tc2VjcmV0'\n") {
		t.Fatalf("manifest did not include encoded data:\n%s", runner.seenManifests[0])
	}
	if !strings.Contains(out.String(), "Generated 1 sealed secret file(s).") {
		t.Fatalf("unexpected output:\n%s", out.String())
	}
}
