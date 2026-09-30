package libvirt

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Loe159/lpic-daily/internal/runner"
)

type ImageCatalog struct {
	SchemaVersion string              `json:"schema_version"`
	Images        []ImageCatalogEntry `json:"images"`
}

type ImageCatalogEntry struct {
	ID            string     `json:"id"`
	RelativePath  string     `json:"relative_path"`
	SHA256        string     `json:"sha256"`
	Format        string     `json:"format"`
	Architecture  string     `json:"architecture"`
	Distribution  string     `json:"distribution"`
	Version       string     `json:"version"`
	VirtualSizeMB int        `json:"virtual_size_mb"`
	Firmware      []string   `json:"firmware"`
	Provenance    Provenance `json:"provenance"`
}

type SourceIntegrity struct {
	Algorithm string `json:"algorithm"`
	Encoding  string `json:"encoding"`
	Value     string `json:"value"`
}

type Provenance struct {
	SourceURL       string          `json:"source_url"`
	SourceIntegrity SourceIntegrity `json:"source_integrity"`
	BuildRecipe     string          `json:"build_recipe"`
	BuiltAt         string          `json:"built_at"`
}

func (integrity SourceIntegrity) Validate() error {
	var expectedLength int
	switch integrity.Algorithm {
	case "sha256":
		expectedLength = 32
	case "sha512":
		expectedLength = 64
	default:
		return fmt.Errorf("unsupported source integrity algorithm %q", integrity.Algorithm)
	}

	var (
		decoded []byte
		err     error
	)
	switch integrity.Encoding {
	case "hex":
		decoded, err = hex.DecodeString(integrity.Value)
	case "base64":
		decoded, err = base64.StdEncoding.Strict().DecodeString(integrity.Value)
	default:
		return fmt.Errorf("unsupported source integrity encoding %q", integrity.Encoding)
	}
	if err != nil {
		return fmt.Errorf("decode source integrity %s: %w", integrity.Encoding, err)
	}
	if len(decoded) != expectedLength {
		return fmt.Errorf(
			"source integrity %s digest has %d bytes, want %d",
			integrity.Algorithm,
			len(decoded),
			expectedLength,
		)
	}
	return nil
}

func LoadImageCatalog(path, imageRoot string) (*ImageCatalog, error) {
	if path == "" || !filepath.IsAbs(path) {
		return nil, errors.New("image catalog path must be absolute")
	}
	if err := pathWithinRoot(imageRoot, path); err != nil {
		return nil, fmt.Errorf("catalog path: %w", err)
	}

	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("open image catalog: %w", err)
	}
	defer file.Close()

	const maxCatalogSize = 2 << 20
	decoder := json.NewDecoder(io.LimitReader(file, maxCatalogSize+1))
	decoder.DisallowUnknownFields()

	var catalog ImageCatalog
	if err := decoder.Decode(&catalog); err != nil {
		return nil, fmt.Errorf("decode image catalog: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("image catalog contains trailing JSON")
		}
		return nil, fmt.Errorf("decode image catalog trailing data: %w", err)
	}
	if catalog.SchemaVersion != "1.0.0" {
		return nil, fmt.Errorf("unsupported image catalog schema version %q", catalog.SchemaVersion)
	}
	if len(catalog.Images) == 0 {
		return nil, errors.New("image catalog must contain at least one image")
	}
	if len(catalog.Images) > 64 {
		return nil, errors.New("image catalog contains too many images")
	}

	seen := make(map[string]struct{}, len(catalog.Images))
	for index := range catalog.Images {
		entry := &catalog.Images[index]
		if _, exists := seen[entry.ID]; exists {
			return nil, fmt.Errorf("duplicate image ID %q", entry.ID)
		}
		seen[entry.ID] = struct{}{}
		if _, err := entry.Descriptor(imageRoot); err != nil {
			return nil, fmt.Errorf("image %s: %w", entry.ID, err)
		}
		if strings.TrimSpace(entry.Version) == "" {
			return nil, fmt.Errorf("image %s: version is required", entry.ID)
		}
		if strings.TrimSpace(entry.Provenance.SourceURL) == "" {
			return nil, fmt.Errorf("image %s: provenance source_url is required", entry.ID)
		}
		if err := entry.Provenance.SourceIntegrity.Validate(); err != nil {
			return nil, fmt.Errorf("image %s: provenance source_integrity: %w", entry.ID, err)
		}
		if strings.TrimSpace(entry.Provenance.BuildRecipe) == "" {
			return nil, fmt.Errorf("image %s: provenance build_recipe is required", entry.ID)
		}
		if strings.TrimSpace(entry.Provenance.BuiltAt) == "" {
			return nil, fmt.Errorf("image %s: provenance built_at is required", entry.ID)
		}
	}
	return &catalog, nil
}

func (catalog *ImageCatalog) Resolve(id, imageRoot string) (ImageDescriptor, error) {
	if catalog == nil {
		return ImageDescriptor{}, errors.New("image catalog is required")
	}
	for _, entry := range catalog.Images {
		if entry.ID == id {
			return entry.Descriptor(imageRoot)
		}
	}
	return ImageDescriptor{}, fmt.Errorf("unknown trusted VM image %q", id)
}

func (entry ImageCatalogEntry) Descriptor(imageRoot string) (ImageDescriptor, error) {
	if filepath.IsAbs(entry.RelativePath) {
		return ImageDescriptor{}, errors.New("relative_path must not be absolute")
	}
	clean := filepath.Clean(entry.RelativePath)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return ImageDescriptor{}, errors.New("relative_path escapes image root")
	}

	firmware := make([]runner.FirmwareMode, 0, len(entry.Firmware))
	for _, value := range entry.Firmware {
		firmware = append(firmware, runner.FirmwareMode(value))
	}

	descriptor := ImageDescriptor{
		ID:            entry.ID,
		Path:          filepath.Join(filepath.Clean(imageRoot), clean),
		SHA256:        entry.SHA256,
		Format:        entry.Format,
		Architecture:  entry.Architecture,
		Distribution:  entry.Distribution,
		VirtualSizeMB: entry.VirtualSizeMB,
		FirmwareModes: firmware,
	}
	if err := descriptor.Validate(imageRoot); err != nil {
		return ImageDescriptor{}, err
	}
	return descriptor, nil
}
