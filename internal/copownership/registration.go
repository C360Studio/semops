package copownership

import (
	"context"
	"fmt"

	"github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/pkg/ownership"
	"github.com/c360studio/semstreams/pkg/projection"
)

type BindingResult struct {
	Incarnation string
	Owners      []string
}

func (r BindingResult) OwnerTokenSuffix() string {
	return r.Incarnation
}

func (r BindingResult) OwnerToken(owner string) string {
	if owner == "" || r.Incarnation == "" {
		return ""
	}
	return owner + "#" + r.Incarnation
}

func RegisterFirstPhase(
	ctx context.Context,
	registry *ownership.Registry,
	heartbeater *ownership.Heartbeater,
) (BindingResult, error) {
	return RegisterOwnedContracts(ctx, registry, heartbeater, cop.FirstPhaseOwnedContracts())
}

func RegisterOwnedContracts(
	ctx context.Context,
	registry *ownership.Registry,
	heartbeater *ownership.Heartbeater,
	owned []cop.OwnedContract,
) (BindingResult, error) {
	return registerOwnedContracts(ctx, registry, heartbeater, owned, projection.BindAndHeartbeat)
}

type bindAndHeartbeatFunc func(
	context.Context,
	*ownership.Registry,
	*ownership.Heartbeater,
	string,
	...projection.Contract,
) error

func registerOwnedContracts(
	ctx context.Context,
	registry *ownership.Registry,
	heartbeater *ownership.Heartbeater,
	owned []cop.OwnedContract,
	bind bindAndHeartbeatFunc,
) (BindingResult, error) {
	if registry == nil {
		return BindingResult{}, fmt.Errorf("register COP ownership: registry is nil")
	}
	if heartbeater == nil {
		return BindingResult{}, fmt.Errorf("register COP ownership: heartbeater is nil")
	}
	if bind == nil {
		return BindingResult{}, fmt.Errorf("register COP ownership: bind function is nil")
	}
	if len(owned) == 0 {
		return BindingResult{}, fmt.Errorf("register COP ownership: no contracts")
	}

	order := make([]string, 0, len(owned))
	groups := make(map[string][]projection.Contract, len(owned))
	for _, item := range owned {
		if item.Owner == "" {
			return BindingResult{}, fmt.Errorf("register COP ownership: contract %q has no owner", item.Contract.Name)
		}
		if _, ok := groups[item.Owner]; !ok {
			order = append(order, item.Owner)
		}
		groups[item.Owner] = append(groups[item.Owner], item.Contract)
	}

	for _, owner := range order {
		if err := bind(ctx, registry, heartbeater, owner, groups[owner]...); err != nil {
			return BindingResult{}, fmt.Errorf("register COP owner %q: %w", owner, err)
		}
	}

	return BindingResult{
		Incarnation: registry.Incarnation(),
		Owners:      append([]string(nil), order...),
	}, nil
}
