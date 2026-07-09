package cop

import (
	"context"
	"time"

	"github.com/c360studio/semops/internal/adapters/semlinkdemo"
)

type SemLinkArtifactProvider struct {
	artifact semlinkdemo.Artifact
	fallback SnapshotProvider
	now      func() time.Time
}

func NewSemLinkArtifactProvider(artifact semlinkdemo.Artifact, fallback SnapshotProvider) *SemLinkArtifactProvider {
	return &SemLinkArtifactProvider{
		artifact: artifact,
		fallback: fallback,
		now:      time.Now,
	}
}

func NewSemLinkArtifactProviderFromFile(
	path string,
	fallback SnapshotProvider,
	opts semlinkdemo.ArtifactOptions,
) (*SemLinkArtifactProvider, error) {
	artifact, err := semlinkdemo.LoadArtifactFile(path, opts)
	if err != nil {
		return nil, err
	}
	return &SemLinkArtifactProvider{
		artifact: artifact,
		fallback: fallback,
		now:      time.Now,
	}, nil
}

func (p *SemLinkArtifactProvider) Snapshot(ctx context.Context) (Snapshot, error) {
	var snapshot Snapshot
	if p.fallback != nil {
		base, err := p.fallback.Snapshot(ctx)
		if err != nil {
			return Snapshot{}, err
		}
		snapshot = base
	} else {
		now := p.artifact.GeneratedAt
		if now.IsZero() && p.now != nil {
			now = p.now().UTC()
		}
		snapshot = Snapshot{
			GeneratedAt: now,
			Scenario:    "semlink-artifact",
		}
	}
	fleet := CompanionFleetFromSemLinkArtifact(p.artifact)
	snapshot.CompanionFleets = replaceCompanionFleet(snapshot.CompanionFleets, fleet)
	snapshot.Summary.ActiveCompanionNodes = activeCompanionNodeCount(snapshot.CompanionFleets)
	if snapshot.GeneratedAt.IsZero() {
		snapshot.GeneratedAt = fleet.UpdatedAt
	}
	return snapshot, nil
}

func replaceCompanionFleet(fleets []CompanionFleet, fleet CompanionFleet) []CompanionFleet {
	next := make([]CompanionFleet, 0, len(fleets)+1)
	replaced := false
	for _, existing := range fleets {
		if existing.ID == fleet.ID {
			next = append(next, fleet)
			replaced = true
			continue
		}
		next = append(next, existing)
	}
	if !replaced {
		next = append(next, fleet)
	}
	return next
}

func activeCompanionNodeCount(fleets []CompanionFleet) int {
	var count int
	for _, fleet := range fleets {
		count += fleet.NodeCount
	}
	return count
}
