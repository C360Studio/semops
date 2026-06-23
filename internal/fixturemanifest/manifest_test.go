package fixturemanifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const manifestPath = "fixtures/manifest.json"

type manifestFile struct {
	Version string          `json:"version"`
	Entries []fixtureRecord `json:"fixtures"`
}

type fixtureRecord struct {
	ID                  string   `json:"id"`
	Feed                string   `json:"feed"`
	Tier                string   `json:"tier"`
	Path                string   `json:"path"`
	SourceURL           string   `json:"source_url"`
	SourceDescription   string   `json:"source_description"`
	CapturedAt          string   `json:"captured_at"`
	DerivedAt           string   `json:"derived_at"`
	SHA256              string   `json:"sha256"`
	SizeBytes           int64    `json:"size_bytes"`
	LicenseOrProvenance string   `json:"license_or_provenance"`
	ClaimScope          string   `json:"claim_scope"`
	Review              string   `json:"review"`
	CommitStatus        string   `json:"commit_status"`
	ObservedFields      []string `json:"observed_fields"`
	SyntheticFields     []string `json:"synthetic_fields"`
}

func TestFixtureManifestDeclaresTiersAndValidatesCommittedArtifacts(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, manifestPath))
	if err != nil {
		t.Fatalf("read fixture manifest: %v", err)
	}

	var manifest manifestFile
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("decode fixture manifest: %v", err)
	}
	if manifest.Version != "semops.fixture-manifest.v1" {
		t.Fatalf("manifest version = %q", manifest.Version)
	}
	if len(manifest.Entries) == 0 {
		t.Fatal("fixture manifest is empty")
	}

	seen := map[string]bool{}
	feeds := map[string]bool{}
	manifestedPaths := map[string]bool{}
	for _, entry := range manifest.Entries {
		validateFixtureEntry(t, root, entry, seen)
		feeds[entry.Feed] = true
		manifestedPaths[filepath.Clean(entry.Path)] = true
	}
	for _, feed := range []string{"cap", "dji", "weather", "adsb", "sapient", "klv"} {
		if !feeds[feed] {
			t.Fatalf("fixture manifest missing feed %q", feed)
		}
	}
	validatePortableFixtureFilesAreManifested(t, root, manifestedPaths)
}

func validateFixtureEntry(t *testing.T, root string, entry fixtureRecord, seen map[string]bool) {
	t.Helper()
	if entry.ID == "" {
		t.Fatal("fixture entry missing id")
	}
	if seen[entry.ID] {
		t.Fatalf("duplicate fixture id %q", entry.ID)
	}
	seen[entry.ID] = true

	required := map[string]string{
		"feed":                  entry.Feed,
		"tier":                  entry.Tier,
		"path":                  entry.Path,
		"sha256":                entry.SHA256,
		"license_or_provenance": entry.LicenseOrProvenance,
		"claim_scope":           entry.ClaimScope,
		"review":                entry.Review,
		"commit_status":         entry.CommitStatus,
	}
	for name, value := range required {
		if value == "" {
			t.Fatalf("fixture %s missing %s", entry.ID, name)
		}
	}
	if entry.SourceURL == "" && entry.SourceDescription == "" {
		t.Fatalf("fixture %s needs source_url or source_description", entry.ID)
	}
	if entry.CapturedAt == "" && entry.DerivedAt == "" {
		t.Fatalf("fixture %s needs captured_at or derived_at", entry.ID)
	}
	if len(entry.ObservedFields) == 0 && len(entry.SyntheticFields) == 0 {
		t.Fatalf("fixture %s must declare observed or synthetic fields", entry.ID)
	}
	if !allowedTier(entry.Tier) {
		t.Fatalf("fixture %s has unknown tier %q", entry.ID, entry.Tier)
	}
	if !allowedCommitStatus(entry.CommitStatus) {
		t.Fatalf("fixture %s has unknown commit_status %q", entry.ID, entry.CommitStatus)
	}
	reviewPath := filepath.Join(root, entry.Review)
	if _, err := os.Stat(reviewPath); err != nil {
		t.Fatalf("fixture %s review %s is not readable: %v", entry.ID, entry.Review, err)
	}

	artifactPath := filepath.Join(root, entry.Path)
	switch entry.CommitStatus {
	case "committed":
		validateFixtureArtifact(t, artifactPath, entry)
	case "ignored_local":
		if _, err := os.Stat(artifactPath); err == nil {
			validateFixtureArtifact(t, artifactPath, entry)
		} else if !os.IsNotExist(err) {
			t.Fatalf("fixture %s ignored artifact stat failed: %v", entry.ID, err)
		}
	case "generated":
		if _, err := os.Stat(artifactPath); err != nil {
			t.Fatalf("fixture %s generated source %s is not readable: %v", entry.ID, entry.Path, err)
		}
	}
}

func validateFixtureArtifact(t *testing.T, path string, entry fixtureRecord) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("fixture %s path %s is not readable: %v", entry.ID, entry.Path, err)
	}
	if int64(len(data)) != entry.SizeBytes {
		t.Fatalf("fixture %s size = %d, want %d", entry.ID, len(data), entry.SizeBytes)
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != entry.SHA256 {
		t.Fatalf("fixture %s sha256 = %s, want %s", entry.ID, got, entry.SHA256)
	}
}

func allowedTier(value string) bool {
	switch value {
	case "ignored_live_capture", "cleared_committed_fixture", "derived_story_fixture":
		return true
	default:
		return false
	}
}

func allowedCommitStatus(value string) bool {
	switch value {
	case "ignored_local", "committed", "generated":
		return true
	default:
		return false
	}
}

func validatePortableFixtureFilesAreManifested(t *testing.T, root string, manifestedPaths map[string]bool) {
	t.Helper()
	fixturesRoot := filepath.Join(root, "fixtures")
	err := filepath.WalkDir(fixturesRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.Clean(rel)
		if d.IsDir() {
			if ignoredLocalFixtureDir(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if ignoredFixtureFile(rel) {
			return nil
		}
		if !manifestedPaths[rel] {
			t.Fatalf("portable fixture file %s missing from %s", rel, manifestPath)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk fixtures: %v", err)
	}
}

func ignoredLocalFixtureDir(rel string) bool {
	switch rel {
	case "fixtures/cap/nws-samples",
		"fixtures/cap/replay",
		"fixtures/cap/schema",
		"fixtures/klv/.cache",
		"fixtures/klv/generated",
		"fixtures/klv/public-samples":
		return true
	default:
		return false
	}
}

func ignoredFixtureFile(rel string) bool {
	base := filepath.Base(rel)
	if strings.HasPrefix(base, ".") {
		return true
	}
	switch rel {
	case manifestPath, "fixtures/klv/README.md":
		return true
	default:
		return false
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		next := filepath.Dir(dir)
		if next == dir {
			t.Fatalf("go.mod not found from %s", dir)
		}
		dir = next
	}
}
