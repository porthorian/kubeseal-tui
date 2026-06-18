package app

import (
	"fmt"
	"path/filepath"
)

func ValidateAndPrepare(cfg *Config) error {
	cfg.SecretName = TrimSpace(cfg.SecretName)
	cfg.SecretType = TrimSpace(cfg.SecretType)
	cfg.ControllerNamespace = TrimSpace(cfg.ControllerNamespace)
	cfg.ControllerName = TrimSpace(cfg.ControllerName)

	if cfg.SecretName == "" {
		return fmt.Errorf("secret name is required")
	}
	if cfg.SecretType == "" {
		cfg.SecretType = "Opaque"
	}
	if cfg.ControllerNamespace == "" {
		return fmt.Errorf("controller namespace cannot be empty")
	}
	if cfg.ControllerName == "" {
		return fmt.Errorf("controller name cannot be empty")
	}
	if len(cfg.Data) == 0 {
		return fmt.Errorf("at least one data entry is required")
	}
	if len(cfg.Targets) == 0 {
		return fmt.Errorf("at least one target is required")
	}

	seenKeys := map[string]struct{}{}
	for i := range cfg.Data {
		entry := &cfg.Data[i]
		key := TrimSpace(entry.Key)
		if key == "" {
			return fmt.Errorf("data key cannot be empty")
		}
		if _, ok := seenKeys[key]; ok {
			return fmt.Errorf("duplicate data key: %s", key)
		}
		entry.Key = key
		seenKeys[key] = struct{}{}
	}

	seenFiles := map[string]struct{}{}
	for i := range cfg.Targets {
		target := &cfg.Targets[i]
		target.Namespace = TrimSpace(target.Namespace)
		if target.Namespace == "" {
			return fmt.Errorf("namespace cannot be empty")
		}
		if target.Dir == "" {
			return fmt.Errorf("output directory cannot be empty for namespace %q", target.Namespace)
		}
		resolved, err := ResolveOutputDir(cfg.CWD, target.Dir)
		if err != nil {
			return fmt.Errorf("output directory does not exist: %s", target.Dir)
		}
		target.Dir = resolved
		target.File = filepath.Join(target.Dir, "sealed-"+cfg.SecretName+".yaml")
		if _, ok := seenFiles[target.File]; ok {
			return fmt.Errorf("duplicate output file path detected: %s", target.File)
		}
		seenFiles[target.File] = struct{}{}
	}

	return nil
}
