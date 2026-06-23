package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateUsesSyntheticLavfiVideoSource(t *testing.T) {
	tempDir := t.TempDir()
	argsPath := filepath.Join(tempDir, "ffmpeg.args")
	ffmpegPath := filepath.Join(tempDir, "ffmpeg")
	writeFakeFFmpeg(t, ffmpegPath)
	t.Setenv("SEMOPS_FAKE_FFMPEG_ARGS", argsPath)

	outPath := filepath.Join(tempDir, "deterministic.ts")
	cfg := config{
		truthPath:  filepath.Join("..", "..", "fixtures", "klv", "misb0601-truth.json"),
		outPath:    outPath,
		ffmpegPath: ffmpegPath,
		overwrite:  false,
		timeout:    time.Second,
	}
	if err := generate(context.Background(), cfg); err != nil {
		t.Fatalf("generate fixture: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("generated output was not created: %v", err)
	}

	argsData, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read fake ffmpeg args: %v", err)
	}
	args := strings.Split(strings.TrimSpace(string(argsData)), "\n")
	requireArgSequence(t, args, "-f", "lavfi", "-i", "testsrc=size=16x16:rate=1")
	requireArgSequence(t, args, "-f", "data")
	requireArgSequence(t, args, "-c:v", "mpeg2video")
	requireArgSequence(t, args, "-c:d", "copy")
	requireArgSequence(t, args, "-f", "mpegts")
}

func writeFakeFFmpeg(t *testing.T, path string) {
	t.Helper()
	script := `#!/bin/sh
set -eu
printf '%s\n' "$@" > "$SEMOPS_FAKE_FFMPEG_ARGS"
last=""
for arg in "$@"; do
  last="$arg"
done
: > "$last"
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake ffmpeg: %v", err)
	}
}

func requireArgSequence(t *testing.T, args []string, want ...string) {
	t.Helper()
	for index := 0; index+len(want) <= len(args); index++ {
		matched := true
		for offset := range want {
			if args[index+offset] != want[offset] {
				matched = false
				break
			}
		}
		if matched {
			return
		}
	}
	t.Fatalf("ffmpeg args missing sequence %q in %q", strings.Join(want, " "), strings.Join(args, " "))
}
