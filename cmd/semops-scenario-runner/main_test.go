package main

import (
	"strings"
	"testing"
	"time"

	semopsapp "github.com/c360studio/semops/internal/app"
	"github.com/c360studio/semops/internal/scenario"
	"github.com/c360studio/semops/pkg/cop"
)

func TestScenarioADSBFixtureEnabledDefaultsFalse(t *testing.T) {
	enabled, err := scenarioADSBFixtureEnabled(func(string) string { return "" })
	if err != nil {
		t.Fatalf("enabled error = %v", err)
	}
	if enabled {
		t.Fatal("ADS-B scenario fixture should default off")
	}
}

func TestScenarioADSBFixtureEnabledParsesBool(t *testing.T) {
	enabled, err := scenarioADSBFixtureEnabled(func(name string) string {
		if name != envScenarioADSBFixture {
			t.Fatalf("env name = %q, want %q", name, envScenarioADSBFixture)
		}
		return "true"
	})
	if err != nil {
		t.Fatalf("enabled error = %v", err)
	}
	if !enabled {
		t.Fatal("ADS-B scenario fixture should parse true")
	}

	_, err = scenarioADSBFixtureEnabled(func(string) string { return "sometimes" })
	if err == nil || !strings.Contains(err.Error(), envScenarioADSBFixture) {
		t.Fatalf("enabled error = %v, want env parse error", err)
	}
}

func TestScenarioModeDefaultsToProductAndParsesContract(t *testing.T) {
	mode, err := scenarioModeFromEnv(func(string) string { return "" })
	if err != nil {
		t.Fatalf("mode error = %v", err)
	}
	if mode != scenarioModeProduct {
		t.Fatalf("default mode = %q, want %q", mode, scenarioModeProduct)
	}
	mode, err = scenarioModeFromEnv(func(name string) string {
		if name != envScenarioMode {
			t.Fatalf("env name = %q, want %q", name, envScenarioMode)
		}
		return "contract"
	})
	if err != nil {
		t.Fatalf("mode error = %v", err)
	}
	if mode != scenarioModeContract {
		t.Fatalf("mode = %q, want %q", mode, scenarioModeContract)
	}
	if _, err := scenarioModeFromEnv(func(string) string { return "graph" }); err == nil {
		t.Fatal("expected invalid scenario mode error")
	}
}

func TestScenarioFeedBoundaryFromEnv(t *testing.T) {
	cfg, err := scenarioFeedBoundaryFromEnv(func(string) string { return "" })
	if err != nil {
		t.Fatalf("boundary error = %v", err)
	}
	if cfg.MAVLinkUDPAddr != "127.0.0.1:14550" ||
		cfg.CoTUDPAddr != "127.0.0.1:18090" ||
		cfg.WriteTimeout != time.Second {
		t.Fatalf("default boundary = %+v", cfg)
	}
	cfg, err = scenarioFeedBoundaryFromEnv(func(name string) string {
		switch name {
		case envScenarioMAVLinkUDPAddr:
			return "semops:14550"
		case envScenarioCoTUDPAddr:
			return "semops:18090"
		case envScenarioWriteTimeout:
			return "1500ms"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatalf("boundary error = %v", err)
	}
	if cfg.MAVLinkUDPAddr != "semops:14550" ||
		cfg.CoTUDPAddr != "semops:18090" ||
		cfg.WriteTimeout != 1500*time.Millisecond {
		t.Fatalf("boundary = %+v", cfg)
	}
	if _, err := scenarioFeedBoundaryFromEnv(func(name string) string {
		if name == envScenarioWriteTimeout {
			return "forever"
		}
		return ""
	}); err == nil {
		t.Fatal("expected invalid timeout error")
	}
}

func TestScenarioFixtureAddsADSBWhenEnabled(t *testing.T) {
	start := time.Date(2026, 6, 20, 16, 0, 0, 0, time.UTC)
	fixture, err := scenarioFixture(start, true)
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	if len(fixture.ADSBSnapshots) != 2 {
		t.Fatalf("ADS-B snapshots = %d, want 2", len(fixture.ADSBSnapshots))
	}
	if fixture.ADSBSnapshots[0].Name != "0001-snapshot" ||
		fixture.ADSBSnapshots[1].Name != "0002-snapshot" {
		t.Fatalf("ADS-B names = %q/%q",
			fixture.ADSBSnapshots[0].Name,
			fixture.ADSBSnapshots[1].Name)
	}
	if fixture.ADSBSnapshots[0].Offset != 20*time.Second ||
		fixture.ADSBSnapshots[1].Offset != 35*time.Second {
		t.Fatalf("ADS-B offsets = %s/%s, want 20s/35s",
			fixture.ADSBSnapshots[0].Offset,
			fixture.ADSBSnapshots[1].Offset)
	}
	if err := fixture.Validate(); err != nil {
		t.Fatalf("fixture should validate: %v", err)
	}
}

func TestScenarioProductFixtureOmitsCAPAndADSB(t *testing.T) {
	fixture, err := scenarioProductFixture(time.Date(2026, 6, 20, 16, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	if len(fixture.MAVLinkFrames) == 0 || len(fixture.CoTEvents) == 0 {
		t.Fatalf("product fixture should keep MAVLink and CoT: %+v", fixture)
	}
	if len(fixture.CAPAlerts) != 0 || len(fixture.ADSBSnapshots) != 0 {
		t.Fatalf("product fixture CAP/ADSB = %d/%d, want 0/0",
			len(fixture.CAPAlerts),
			len(fixture.ADSBSnapshots))
	}
}

func TestComposeProductRunnerDoesNotRequireNATSOrOwnerBindings(t *testing.T) {
	cfg, err := semopsapp.ConfigFromEnv(func(string) string { return "" })
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	client, stopOwners, runner, err := composeProductRunner(cfg, false, scenarioFeedBoundary{
		MAVLinkUDPAddr: "127.0.0.1:14550",
		CoTUDPAddr:     "127.0.0.1:18090",
		WriteTimeout:   time.Second,
	})
	if err != nil {
		t.Fatalf("compose product runner: %v", err)
	}
	if client != nil {
		t.Fatal("product scenario runner must not create a SemStreams NATS client")
	}
	if stopOwners == nil {
		t.Fatal("product scenario runner should return a no-op owner closer")
	}
	if runner == nil {
		t.Fatal("product scenario runner is nil")
	}
	if status := runner.Status(); status.ScenarioID == "" ||
		status.IngressMode != scenario.IngressModeFeedBoundary ||
		status.Summary.CAPAlerts != 0 {
		t.Fatalf("product runner status = %+v", status)
	}
	if _, _, _, err := composeProductRunner(cfg, true, scenarioFeedBoundary{
		MAVLinkUDPAddr: "127.0.0.1:14550",
		CoTUDPAddr:     "127.0.0.1:18090",
		WriteTimeout:   time.Second,
	}); err == nil {
		t.Fatal("expected product runner to reject ADS-B direct scenario fixture")
	}
}

func TestScenarioFixtureLeavesADSBOutByDefault(t *testing.T) {
	fixture, err := scenarioFixture(time.Date(2026, 6, 20, 16, 0, 0, 0, time.UTC), false)
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	if len(fixture.ADSBSnapshots) != 0 {
		t.Fatalf("ADS-B snapshots = %d, want 0", len(fixture.ADSBSnapshots))
	}
}

func TestScenarioOwnedContractsAddsADSBOnlyWhenEnabled(t *testing.T) {
	if hasScenarioOwner(scenarioOwnedContracts(false), cop.OwnerADSB) {
		t.Fatal("ADS-B owner should not be registered for the default scenario path")
	}
	if !hasScenarioOwner(scenarioOwnedContracts(true), cop.OwnerADSB) {
		t.Fatal("ADS-B owner should be registered when ADS-B fixture replay is enabled")
	}
}

func hasScenarioOwner(owned []cop.OwnedContract, owner string) bool {
	for _, item := range owned {
		if item.Owner == owner {
			return true
		}
	}
	return false
}
