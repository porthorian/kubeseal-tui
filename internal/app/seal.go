package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type KubesealRunner interface {
	Seal(ctx context.Context, opts KubesealOptions, secretYAML []byte) ([]byte, error)
}

type KubesealOptions struct {
	ControllerNamespace string
	ControllerName      string
}

type ExecKubesealRunner struct {
	Path string
}

func (r ExecKubesealRunner) CheckAvailable() error {
	path := r.Path
	if path == "" {
		path = "kubeseal"
	}
	if _, err := exec.LookPath(path); err != nil {
		return fmt.Errorf("kubeseal is required but not found in PATH")
	}
	return nil
}

func (r ExecKubesealRunner) Seal(ctx context.Context, opts KubesealOptions, secretYAML []byte) ([]byte, error) {
	path := r.Path
	if path == "" {
		path = "kubeseal"
	}

	tmp, err := os.CreateTemp("", "kubeseal-tui-secret-*.yaml")
	if err != nil {
		return nil, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(secretYAML); err != nil {
		tmp.Close()
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(
		ctx,
		path,
		"--controller-namespace", opts.ControllerNamespace,
		"--controller-name", opts.ControllerName,
		"-f", tmpName,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := stderr.String()
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("kubeseal failed: %s", msg)
	}
	return out, nil
}

func ExistingOutputFiles(targets []Target) []string {
	var existing []string
	for _, target := range targets {
		if _, err := os.Stat(target.File); err == nil {
			existing = append(existing, target.File)
		}
	}
	return existing
}

func SealAndWriteAll(ctx context.Context, cfg Config, runner KubesealRunner, out io.Writer) error {
	encoded, err := EncodedData(cfg.Data)
	if err != nil {
		return err
	}

	for _, target := range cfg.Targets {
		manifest, err := BuildSecretManifest(cfg, target.Namespace, encoded)
		if err != nil {
			return err
		}
		sealed, err := runner.Seal(ctx, KubesealOptions{
			ControllerNamespace: cfg.ControllerNamespace,
			ControllerName:      cfg.ControllerName,
		}, manifest)
		if err != nil {
			return fmt.Errorf("kubeseal failed for namespace %q: %w", target.Namespace, err)
		}
		if err := writeFileAtomic(target.File, sealed); err != nil {
			return err
		}
		fmt.Fprintf(out, "Wrote namespace %q -> %s\n", target.Namespace, target.File)
	}

	fmt.Fprintf(out, "Generated %d sealed secret file(s).\n", len(cfg.Targets))
	return nil
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".kubeseal-tui-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
