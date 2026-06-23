package copownership

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/pkg/ownership"
	"github.com/c360studio/semstreams/pkg/projection"
)

func TestRegisterFirstPhaseBindsAndEnrollsOwners(t *testing.T) {
	ctx := context.Background()
	registry := ownership.NewRegistry(nil, nil, nil)
	heartbeater := registry.NewHeartbeater(time.Hour)
	var calls []string

	result, err := registerOwnedContracts(
		ctx,
		registry,
		heartbeater,
		cop.FirstPhaseOwnedContracts(),
		func(
			gotCtx context.Context,
			gotRegistry *ownership.Registry,
			gotHeartbeater *ownership.Heartbeater,
			owner string,
			contracts ...projection.Contract,
		) (ownership.OwnerToken, error) {
			if gotCtx != ctx {
				t.Fatal("bind received unexpected context")
			}
			if gotRegistry != registry {
				t.Fatal("bind received unexpected registry")
			}
			if gotHeartbeater != heartbeater {
				t.Fatal("bind received unexpected heartbeater")
			}
			registration, err := projection.Derive(owner, contracts...)
			if err != nil {
				t.Fatalf("derive %s: %v", owner, err)
			}
			if len(registration.Claims) == 0 && len(registration.ForeignEdges) == 0 {
				t.Fatalf("owner %s derived no claims", owner)
			}
			if owner == cop.OwnerMAVLink && len(registration.ForeignEdges) != 2 {
				t.Fatalf("MAVLink foreign-edge claims = %d, want 2", len(registration.ForeignEdges))
			}
			heartbeater.Add(owner)
			calls = append(calls, owner)
			return gotRegistry.OwnerToken(owner), nil
		},
	)
	if err != nil {
		t.Fatalf("register first-phase contracts: %v", err)
	}

	wantOwners := []string{
		cop.OwnerAsset,
		cop.OwnerMAVLink,
		cop.OwnerTAK,
		cop.OwnerCAP,
		cop.OwnerFusion,
	}
	if !reflect.DeepEqual(calls, wantOwners) {
		t.Fatalf("bind calls = %#v, want %#v", calls, wantOwners)
	}
	if !reflect.DeepEqual(result.Owners, wantOwners) {
		t.Fatalf("result owners = %#v, want %#v", result.Owners, wantOwners)
	}
	if result.Incarnation == "" {
		t.Fatal("registration result must expose registry incarnation")
	}
	if got, want := result.OwnerToken(cop.OwnerMAVLink).Wire(), registry.OwnerToken(cop.OwnerMAVLink).Wire(); got != want {
		t.Fatalf("MAVLink owner token = %q, want %q", got, want)
	}
	tokenMap := result.OwnerTokenMap()
	tokenMap[cop.OwnerMAVLink] = ownership.OwnerToken{}
	if result.OwnerToken(cop.OwnerMAVLink).IsZero() {
		t.Fatal("owner token map must be a defensive copy")
	}
	for _, owner := range wantOwners {
		if !heartbeater.IsEnrolled(owner) {
			t.Fatalf("owner %s was not enrolled for heartbeat", owner)
		}
	}
}

func TestRegisterOwnedContractsGroupsMultipleContractsForSameOwner(t *testing.T) {
	registry := ownership.NewRegistry(nil, nil, nil)
	heartbeater := registry.NewHeartbeater(time.Hour)
	owned := []cop.OwnedContract{
		{Owner: cop.OwnerMAVLink, Contract: cop.SourceAssetContract()},
		{Owner: cop.OwnerMAVLink, Contract: cop.MAVLinkTrackContract()},
	}

	var gotContractCount int
	result, err := registerOwnedContracts(
		context.Background(),
		registry,
		heartbeater,
		owned,
		func(
			_ context.Context,
			_ *ownership.Registry,
			hb *ownership.Heartbeater,
			owner string,
			contracts ...projection.Contract,
		) (ownership.OwnerToken, error) {
			if owner != cop.OwnerMAVLink {
				t.Fatalf("owner = %q, want %q", owner, cop.OwnerMAVLink)
			}
			gotContractCount = len(contracts)
			if _, err := projection.Derive(owner, contracts...); err != nil {
				t.Fatalf("derive grouped contracts: %v", err)
			}
			hb.Add(owner)
			return registry.OwnerToken(owner), nil
		},
	)
	if err != nil {
		t.Fatalf("register grouped contracts: %v", err)
	}
	if gotContractCount != 2 {
		t.Fatalf("contracts bound = %d, want 2", gotContractCount)
	}
	if !reflect.DeepEqual(result.Owners, []string{cop.OwnerMAVLink}) {
		t.Fatalf("owners = %#v, want single MAVLink owner", result.Owners)
	}
}

func TestRegisterOwnedContractsRejectsMissingRuntimePieces(t *testing.T) {
	registry := ownership.NewRegistry(nil, nil, nil)
	heartbeater := registry.NewHeartbeater(time.Hour)
	contracts := cop.FirstPhaseOwnedContracts()

	tests := []struct {
		name        string
		registry    *ownership.Registry
		heartbeater *ownership.Heartbeater
		owned       []cop.OwnedContract
		bind        bindAndHeartbeatFunc
	}{
		{name: "registry", heartbeater: heartbeater, owned: contracts, bind: noopBind},
		{name: "heartbeater", registry: registry, owned: contracts, bind: noopBind},
		{name: "contracts", registry: registry, heartbeater: heartbeater, bind: noopBind},
		{name: "bind", registry: registry, heartbeater: heartbeater, owned: contracts},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := registerOwnedContracts(
				context.Background(),
				tt.registry,
				tt.heartbeater,
				tt.owned,
				tt.bind,
			); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestRegisterOwnedContractsWrapsBindFailure(t *testing.T) {
	registry := ownership.NewRegistry(nil, nil, nil)
	heartbeater := registry.NewHeartbeater(time.Hour)
	bindErr := errors.New("overlap")

	_, err := registerOwnedContracts(
		context.Background(),
		registry,
		heartbeater,
		[]cop.OwnedContract{{Owner: cop.OwnerMAVLink, Contract: cop.MAVLinkTrackContract()}},
		func(
			context.Context,
			*ownership.Registry,
			*ownership.Heartbeater,
			string,
			...projection.Contract,
		) (ownership.OwnerToken, error) {
			return ownership.OwnerToken{}, bindErr
		},
	)
	if !errors.Is(err, bindErr) {
		t.Fatalf("error = %v, want wrapped bind error", err)
	}
}

func TestRegisterOwnedContractsRejectsZeroOwnerToken(t *testing.T) {
	registry := ownership.NewRegistry(nil, nil, nil)
	heartbeater := registry.NewHeartbeater(time.Hour)

	_, err := registerOwnedContracts(
		context.Background(),
		registry,
		heartbeater,
		[]cop.OwnedContract{{Owner: cop.OwnerMAVLink, Contract: cop.MAVLinkTrackContract()}},
		func(
			context.Context,
			*ownership.Registry,
			*ownership.Heartbeater,
			string,
			...projection.Contract,
		) (ownership.OwnerToken, error) {
			return ownership.OwnerToken{}, nil
		},
	)
	if err == nil || !strings.Contains(err.Error(), "zero owner token") {
		t.Fatalf("error = %v, want zero owner token rejection", err)
	}
}

func noopBind(
	context.Context,
	*ownership.Registry,
	*ownership.Heartbeater,
	string,
	...projection.Contract,
) (ownership.OwnerToken, error) {
	return ownership.ExpectedOwnerToken("noop", "test"), nil
}
