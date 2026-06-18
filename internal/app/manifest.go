package app

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

func EncodedData(entries []DataEntry) ([]string, error) {
	encoded := make([]string, 0, len(entries))
	for _, entry := range entries {
		var payload []byte
		switch entry.Source {
		case DataSourceFile:
			data, err := os.ReadFile(entry.Payload)
			if err != nil {
				return nil, fmt.Errorf("data file is no longer readable for key %q: %s", entry.Key, entry.Payload)
			}
			payload = data
		case DataSourceValue:
			payload = []byte(entry.Payload)
		default:
			return nil, fmt.Errorf("unknown data source for key %q: %s", entry.Key, entry.Source)
		}
		encoded = append(encoded, base64.StdEncoding.EncodeToString(payload))
	}
	return encoded, nil
}

func BuildSecretManifest(cfg Config, namespace string, encodedValues []string) ([]byte, error) {
	if len(encodedValues) != len(cfg.Data) {
		return nil, fmt.Errorf("encoded value count does not match data entries")
	}

	var b strings.Builder
	b.WriteString("apiVersion: v1\n")
	b.WriteString("kind: Secret\n")
	b.WriteString("metadata:\n")
	fmt.Fprintf(&b, "  name: %s\n", yamlQuote(cfg.SecretName))
	fmt.Fprintf(&b, "  namespace: %s\n", yamlQuote(namespace))
	fmt.Fprintf(&b, "type: %s\n", yamlQuote(cfg.SecretType))
	b.WriteString("data:\n")
	for i, entry := range cfg.Data {
		fmt.Fprintf(&b, "  %s: %s\n", yamlQuote(entry.Key), yamlQuote(encodedValues[i]))
	}
	return []byte(b.String()), nil
}

func yamlQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
