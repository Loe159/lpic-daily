package libvirt

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadImageCatalogResolvesOnlyRelativeTrustedPaths(t *testing.T) {
	root := t.TempDir()
	imageRoot := filepath.Join(root, "images")
	if err := os.MkdirAll(filepath.Join(imageRoot, "fedora-44"), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	content := []byte("catalog-base-image")
	digest := sha256.Sum256(content)
	imagePath := filepath.Join(imageRoot, "fedora-44", "base.qcow2")
	if err := os.WriteFile(imagePath, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	catalogPath := filepath.Join(imageRoot, "catalog.json")
	catalogJSON := `{
  "schema_version": "1.0.0",
  "images": [{
    "id": "fedora-44-x86_64-v1",
    "relative_path": "fedora-44/base.qcow2",
    "sha256": "` + hex.EncodeToString(digest[:]) + `",
    "format": "qcow2",
    "architecture": "x86_64",
    "distribution": "fedora",
    "version": "44",
    "virtual_size_mb": 8192,
    "firmware": ["bios", "uefi"],
    "provenance": {
      "source_url": "https://example.invalid/fedora-44.qcow2",
      "source_integrity": {
        "algorithm": "sha256",
        "encoding": "hex",
        "value": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
      },
      "build_recipe": "images/fedora-44.pkr.hcl",
      "built_at": "2026-09-27T00:00:00Z"
    }
  }]
}`
	if err := os.WriteFile(catalogPath, []byte(catalogJSON), 0o600); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	catalog, err := LoadImageCatalog(catalogPath, imageRoot)
	if err != nil {
		t.Fatalf("LoadImageCatalog() error = %v", err)
	}
	image, err := catalog.Resolve("fedora-44-x86_64-v1", imageRoot)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if image.Path != imagePath {
		t.Fatalf("resolved path = %q, want %q", image.Path, imagePath)
	}
	if !image.SupportsFirmware("uefi") {
		t.Fatal("resolved image lost UEFI support")
	}
}

func TestCatalogRejectsTraversalAndAbsolutePaths(t *testing.T) {
	root := t.TempDir()
	imageRoot := filepath.Join(root, "images")
	if err := os.MkdirAll(imageRoot, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	for _, relative := range []string{"../outside.qcow2", "/tmp/outside.qcow2"} {
		catalogPath := filepath.Join(imageRoot, "catalog.json")
		payload := `{
  "schema_version": "1.0.0",
  "images": [{
    "id": "fedora-44-x86_64-v1",
    "relative_path": "` + relative + `",
    "sha256": "` + strings.Repeat("a", 64) + `",
    "format": "qcow2",
    "architecture": "x86_64",
    "distribution": "fedora",
    "version": "44",
    "virtual_size_mb": 8192,
    "firmware": ["bios"],
    "provenance": {
      "source_url": "https://example.invalid/image",
      "source_integrity": {
        "algorithm": "sha256",
        "encoding": "hex",
        "value": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
      },
      "build_recipe": "recipe",
      "built_at": "2026-09-27T00:00:00Z"
    }
  }]
}`
		if err := os.WriteFile(catalogPath, []byte(payload), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		if _, err := LoadImageCatalog(catalogPath, imageRoot); err == nil {
			t.Fatalf("relative_path %q unexpectedly accepted", relative)
		}
	}
}

func TestCatalogRejectsUnknownFieldsAndDuplicateIDs(t *testing.T) {
	root := t.TempDir()
	imageRoot := filepath.Join(root, "images")
	if err := os.MkdirAll(imageRoot, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	catalogPath := filepath.Join(imageRoot, "catalog.json")
	if err := os.WriteFile(catalogPath, []byte(`{"schema_version":"1.0.0","images":[],"unexpected":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := LoadImageCatalog(catalogPath, imageRoot); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown field error = %v", err)
	}
}
