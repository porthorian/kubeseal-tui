package app

import (
	"fmt"
	"os"
	"strings"
)

func TrimSpace(s string) string {
	return strings.TrimSpace(s)
}

func ParseDataFlag(cfg *Config, raw string, source DataSource) error {
	key, value, ok := strings.Cut(raw, "=")
	if !ok {
		return fmt.Errorf("expected KEY=VALUE format, got: %s", raw)
	}

	key = TrimSpace(key)
	if key == "" {
		return fmt.Errorf("data key cannot be empty in: %s", raw)
	}

	switch source {
	case DataSourceValue:
		return AddDataValue(cfg, key, value)
	case DataSourceFile:
		return AddDataFile(cfg, key, TrimSpace(value))
	default:
		return fmt.Errorf("unknown data source: %s", source)
	}
}

func AddDataValue(cfg *Config, key, value string) error {
	if TrimSpace(key) == "" {
		return fmt.Errorf("data key cannot be empty")
	}
	if dataKeyExists(cfg.Data, key) {
		return fmt.Errorf("duplicate data key: %s", key)
	}
	cfg.Data = append(cfg.Data, DataEntry{Key: key, Source: DataSourceValue, Payload: value})
	return nil
}

func AddDataFile(cfg *Config, key, path string) error {
	if TrimSpace(key) == "" {
		return fmt.Errorf("data key cannot be empty")
	}
	if path == "" {
		return fmt.Errorf("file path for key %q cannot be empty", key)
	}
	if dataKeyExists(cfg.Data, key) {
		return fmt.Errorf("duplicate data key: %s", key)
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file for key %q is not readable: %s", key, path)
		}
		return fmt.Errorf("file for key %q is not readable: %s: %w", key, path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("file for key %q is not readable: %s", key, path)
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("file for key %q is not readable: %s: %w", key, path, err)
	}
	if err := file.Close(); err != nil {
		return err
	}
	cfg.Data = append(cfg.Data, DataEntry{Key: key, Source: DataSourceFile, Payload: path})
	return nil
}

func ParseTargetFlag(cfg *Config, raw string) error {
	namespace, outputDir, ok := strings.Cut(raw, "=")
	if !ok {
		return fmt.Errorf("expected NAMESPACE=OUTPUT_DIR format, got: %s", raw)
	}
	return AddTarget(cfg, namespace, outputDir)
}

func AddTarget(cfg *Config, namespace, outputDirInput string) error {
	namespace = TrimSpace(namespace)
	outputDirInput = TrimSpace(outputDirInput)
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	if outputDirInput == "" {
		return fmt.Errorf("output directory cannot be empty for namespace %q", namespace)
	}
	dir, err := ResolveOutputDir(cfg.CWD, outputDirInput)
	if err != nil {
		return fmt.Errorf("output directory does not exist: %s", outputDirInput)
	}
	cfg.Targets = append(cfg.Targets, Target{Namespace: namespace, Dir: dir})
	return nil
}

func dataKeyExists(entries []DataEntry, key string) bool {
	for _, entry := range entries {
		if entry.Key == key {
			return true
		}
	}
	return false
}
