package cap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	envCAPXSDPath           = "SEMOPS_CAP_XSD_PATH"
	envCAPSchemaSamplePaths = "SEMOPS_CAP_SCHEMA_SAMPLE_PATHS"
	envCAPSchemaReplayPath  = "SEMOPS_CAP_SCHEMA_REPLAY_PATH"
)

func TestCAPSchemaSmokeWithLocalSamples(t *testing.T) {
	schemaPath := strings.TrimSpace(os.Getenv(envCAPXSDPath))
	if schemaPath == "" {
		t.Skipf("set %s to run CAP 1.2 XSD validation against local samples", envCAPXSDPath)
	}
	schemaPath, err := resolveLocalPath(schemaPath)
	if err != nil {
		t.Fatalf("resolve %s: %v", envCAPXSDPath, err)
	}

	xmllint, err := exec.LookPath("xmllint")
	if err != nil {
		t.Fatalf("%s requires xmllint on PATH: %v", envCAPXSDPath, err)
	}
	samples, err := capSchemaSamples()
	if err != nil {
		t.Fatalf("collect CAP schema samples: %v", err)
	}
	if len(samples) == 0 {
		t.Fatalf(
			"set %s or %s with local CAP XML files or replay JSONL records",
			envCAPSchemaSamplePaths,
			envCAPSchemaReplayPath,
		)
	}

	tempDir := t.TempDir()
	for _, sample := range samples {
		t.Run(sample.name, func(t *testing.T) {
			if _, err := Parse(sample.rawXML); err != nil {
				t.Fatalf("SemOps CAP parse failed before schema validation: %v", err)
			}
			xmlPath := sample.path
			if xmlPath == "" {
				xmlPath = filepath.Join(tempDir, sanitizeCAPSchemaSampleName(sample.name)+".xml")
				if err := os.WriteFile(xmlPath, sample.rawXML, 0o644); err != nil {
					t.Fatalf("write replay sample XML: %v", err)
				}
			}
			validateCAPXMLWithXSD(t, xmllint, schemaPath, xmlPath)
		})
	}
}

type capSchemaSample struct {
	name   string
	path   string
	rawXML []byte
}

func capSchemaSamples() ([]capSchemaSample, error) {
	var samples []capSchemaSample
	pathsEnv := strings.TrimSpace(os.Getenv(envCAPSchemaSamplePaths))
	if pathsEnv != "" {
		paths, err := expandCAPSamplePaths(pathsEnv)
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			rawXML, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("read sample %s: %w", path, err)
			}
			samples = append(samples, capSchemaSample{
				name:   filepath.Base(path),
				path:   path,
				rawXML: rawXML,
			})
		}
	}

	replayPath := strings.TrimSpace(os.Getenv(envCAPSchemaReplayPath))
	if replayPath != "" {
		replayPath, err := resolveLocalPath(replayPath)
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", envCAPSchemaReplayPath, err)
		}
		records, err := LoadReplay(replayPath)
		if err != nil {
			return nil, fmt.Errorf("load replay %s: %w", replayPath, err)
		}
		for index, record := range records {
			name := strings.TrimSpace(record.Ref)
			if name == "" {
				name = fmt.Sprintf("replay-record-%04d", index+1)
			}
			samples = append(samples, capSchemaSample{
				name:   name,
				rawXML: append([]byte(nil), record.RawXML...),
			})
		}
	}
	return samples, nil
}

func expandCAPSamplePaths(pathsEnv string) ([]string, error) {
	var out []string
	for _, token := range filepath.SplitList(pathsEnv) {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		expanded, err := expandCAPSamplePathToken(token)
		if err != nil {
			return nil, err
		}
		out = append(out, expanded...)
	}
	return out, nil
}

func expandCAPSamplePathToken(token string) ([]string, error) {
	candidates, err := candidateLocalPaths(token)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, candidate := range candidates {
		if hasGlobMeta(candidate) {
			matches, err := filepath.Glob(candidate)
			if err != nil {
				return nil, fmt.Errorf("expand glob %s: %w", token, err)
			}
			for _, match := range matches {
				files, err := capXMLFiles(match)
				if err != nil {
					return nil, err
				}
				out = append(out, files...)
			}
			continue
		}
		files, err := capXMLFiles(candidate)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		out = append(out, files...)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no CAP XML samples matched %q", token)
	}
	return dedupeStrings(out), nil
}

func capXMLFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		return []string{abs}, nil
	}
	var files []string
	err = filepath.WalkDir(path, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".xml") {
			abs, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			files = append(files, abs)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk CAP sample dir %s: %w", path, err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("CAP sample dir %s contains no .xml files", path)
	}
	return files, nil
}

func validateCAPXMLWithXSD(t *testing.T, xmllint, schemaPath, xmlPath string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, xmllint, "--noout", "--schema", schemaPath, xmlPath)
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("xmllint timed out for %s", xmlPath)
	}
	if err != nil {
		t.Fatalf("xmllint schema validation failed for %s: %v\n%s", xmlPath, err, strings.TrimSpace(string(output)))
	}
}

func resolveLocalPath(path string) (string, error) {
	candidates, err := candidateLocalPaths(path)
	if err != nil {
		return "", err
	}
	for _, candidate := range candidates {
		if hasGlobMeta(candidate) {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Abs(candidate)
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("path does not exist: %s", path)
}

func candidateLocalPaths(path string) ([]string, error) {
	if filepath.IsAbs(path) {
		return []string{path}, nil
	}
	candidates := []string{path}
	root, err := repoRoot()
	if err != nil {
		return candidates, nil
	}
	repoPath := filepath.Join(root, path)
	if repoPath != path {
		candidates = append(candidates, repoPath)
	}
	return candidates, nil
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		next := filepath.Dir(dir)
		if next == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = next
	}
}

func hasGlobMeta(path string) bool {
	return strings.ContainsAny(path, "*?[")
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func sanitizeCAPSchemaSampleName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "cap-sample"
	}
	return out
}
