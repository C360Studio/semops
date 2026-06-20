package main

import (
	"strings"
	"testing"
	"time"

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
