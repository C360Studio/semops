package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/c360studio/semops/internal/egress/semconnect"
)

const defaultTimeout = 15 * time.Second

type config struct {
	BaseURL   string
	DryRun    bool
	Timeout   time.Duration
	UserAgent string
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	cfg := config{
		BaseURL:   strings.TrimSpace(os.Getenv("SEMOPS_SEMCONNECT_BASE_URL")),
		Timeout:   defaultTimeout,
		UserAgent: "semops-semconnect-fixture/0.1",
	}
	fs := flag.NewFlagSet("semops-semconnect-fixture", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&cfg.BaseURL, "base-url", cfg.BaseURL, "SemConnect CS API base URL")
	fs.BoolVar(&cfg.DryRun, "dry-run", cfg.DryRun, "print deterministic read-side request-plan evidence without sending HTTP requests")
	fs.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "fixture execution timeout")
	fs.StringVar(&cfg.UserAgent, "user-agent", cfg.UserAgent, "User-Agent for SemConnect HTTP requests")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var result semconnect.ReadSideFixtureResult
	var err error
	if cfg.DryRun {
		result, err = semconnect.PlanReadSideFixture()
	} else {
		if strings.TrimSpace(cfg.BaseURL) == "" {
			return errors.New("base URL required; pass -base-url or set SEMOPS_SEMCONNECT_BASE_URL")
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
		defer cancel()
		result, err = semconnect.RunReadSideFixture(ctx, cfg.BaseURL,
			semconnect.WithFixtureExecuteOptions(semconnect.WithUserAgent(cfg.UserAgent)),
		)
	}
	if encodeErr := json.NewEncoder(stdout).Encode(result); encodeErr != nil {
		return fmt.Errorf("encode fixture evidence: %w", encodeErr)
	}
	return err
}
