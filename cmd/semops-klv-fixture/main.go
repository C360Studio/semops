package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	klvcomponent "github.com/c360studio/semops/internal/components/klv"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	cfg := parseConfig()
	ctx, cancel := context.WithTimeout(context.Background(), fixtureTimeout(cfg.timeout))
	defer cancel()
	if err := generate(ctx, cfg); err != nil {
		log.Fatalf("generate KLV fixture: %v", err)
	}
	log.Printf(
		"SemOps KLV fixture v%s (commit: %s, built: %s) wrote %s",
		version,
		commit,
		buildDate,
		cfg.outPath,
	)
}

type config struct {
	truthPath  string
	outPath    string
	ffmpegPath string
	overwrite  bool
	timeout    time.Duration
}

func parseConfig() config {
	cfg := config{}
	flag.StringVar(&cfg.truthPath, "truth", "fixtures/klv/misb0601-truth.json", "MISB ST 0601 truth JSON path")
	flag.StringVar(&cfg.outPath, "out", "fixtures/klv/generated/deterministic.ts", "MPEG-TS output path")
	flag.StringVar(&cfg.ffmpegPath, "ffmpeg", klvcomponent.DefaultFFmpegPath, "ffmpeg executable path")
	flag.BoolVar(&cfg.overwrite, "overwrite", false, "overwrite the output fixture if it already exists")
	flag.DurationVar(&cfg.timeout, "timeout", 15*time.Second, "fixture generation timeout")
	flag.Parse()
	return cfg
}

func fixtureTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return 15 * time.Second
	}
	return timeout
}

func generate(ctx context.Context, cfg config) error {
	if cfg.truthPath == "" {
		return fmt.Errorf("truth path is required")
	}
	if cfg.outPath == "" {
		return fmt.Errorf("output path is required")
	}
	if cfg.ffmpegPath == "" {
		return fmt.Errorf("ffmpeg path is required")
	}
	if !cfg.overwrite {
		if _, err := os.Stat(cfg.outPath); err == nil {
			return fmt.Errorf("output %s already exists; pass -overwrite to replace it", cfg.outPath)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat output %s: %w", cfg.outPath, err)
		}
	}

	truth, err := readTruth(cfg.truthPath)
	if err != nil {
		return err
	}
	packetBytes, err := klvcomponent.EncodeMISB0601Truth(truth)
	if err != nil {
		return fmt.Errorf("encode KLV truth packet: %w", err)
	}
	outDir := filepath.Dir(cfg.outPath)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create output directory %s: %w", outDir, err)
	}
	tempDir, err := os.MkdirTemp(outDir, ".semops-klv-fixture-*")
	if err != nil {
		return fmt.Errorf("create temp fixture directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	packetPath := filepath.Join(tempDir, "deterministic.klv")
	if err := os.WriteFile(packetPath, packetBytes, 0o600); err != nil {
		return fmt.Errorf("write packet fixture: %w", err)
	}
	if err := muxFixture(ctx, cfg.ffmpegPath, packetPath, cfg.outPath); err != nil {
		return err
	}
	return nil
}

func readTruth(path string) (klvcomponent.MISB0601Truth, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return klvcomponent.MISB0601Truth{}, fmt.Errorf("read truth fixture %s: %w", path, err)
	}
	var truth klvcomponent.MISB0601Truth
	if err := json.Unmarshal(data, &truth); err != nil {
		return klvcomponent.MISB0601Truth{}, fmt.Errorf("parse truth fixture %s: %w", path, err)
	}
	if err := truth.Validate(); err != nil {
		return klvcomponent.MISB0601Truth{}, fmt.Errorf("validate truth fixture %s: %w", path, err)
	}
	return truth, nil
}

func muxFixture(ctx context.Context, ffmpegPath string, packetPath string, outPath string) error {
	cmd := exec.CommandContext(
		ctx,
		ffmpegPath,
		"-v", "error",
		"-y",
		"-f", "lavfi",
		"-i", "testsrc=size=16x16:rate=1",
		"-f", "data",
		"-i", packetPath,
		"-map", "0:v:0",
		"-map", "1:0",
		"-c:v", "mpeg2video",
		"-c:d", "copy",
		"-t", "1",
		"-shortest",
		"-f", "mpegts",
		outPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mux deterministic MPEG-TS fixture: %w; output=%s", err, output)
	}
	return nil
}
