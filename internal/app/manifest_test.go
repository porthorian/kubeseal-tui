package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncodedDataReadsValueAndFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "password.txt")
	if err := os.WriteFile(path, []byte("file-secret\n"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	encoded, err := EncodedData([]DataEntry{
		{Key: "token", Source: DataSourceValue, Payload: "plain-secret"},
		{Key: "password", Source: DataSourceFile, Payload: path},
	})
	if err != nil {
		t.Fatalf("EncodedData returned error: %v", err)
	}

	if encoded[0] != "cGxhaW4tc2VjcmV0" {
		t.Fatalf("unexpected encoded inline value: %q", encoded[0])
	}
	if encoded[1] != "ZmlsZS1zZWNyZXQK" {
		t.Fatalf("unexpected encoded file value: %q", encoded[1])
	}
}

func TestBuildSecretManifestQuotesValues(t *testing.T) {
	cfg := Config{
		SecretName: "api's-secret",
		SecretType: "Opaque",
		Data: []DataEntry{
			{Key: "token", Source: DataSourceValue, Payload: "ignored"},
		},
	}

	manifest, err := BuildSecretManifest(cfg, "prod", []string{"dmFsdWU="})
	if err != nil {
		t.Fatalf("BuildSecretManifest returned error: %v", err)
	}

	want := "apiVersion: v1\n" +
		"kind: Secret\n" +
		"metadata:\n" +
		"  name: 'api''s-secret'\n" +
		"  namespace: 'prod'\n" +
		"type: 'Opaque'\n" +
		"data:\n" +
		"  'token': 'dmFsdWU='\n"
	if string(manifest) != want {
		t.Fatalf("unexpected manifest:\n%s", string(manifest))
	}
}
