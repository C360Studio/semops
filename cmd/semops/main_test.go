package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	copapi "github.com/c360studio/semops/internal/api/cop"
	semopsapp "github.com/c360studio/semops/internal/app"
	commandprojector "github.com/c360studio/semops/internal/projectors/command"
	copmodel "github.com/c360studio/semops/pkg/cop"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/pkg/ownership"
)

func TestSemLinkReadbackHandlerOptionWiresGraphBackedIngress(t *testing.T) {
	cfg := semopsapp.DefaultConfig()
	cfg.COP.SemLinkReadbackEnabled = true
	cfg.COP.OperatorIdentityMode = semopsapp.COPOperatorIdentityModeTrustedHeaders
	cfg.COP.GraphQueryTimeout = 25 * time.Millisecond
	cfg.COP.SemLinkReadbackWriteTimeout = 30 * time.Millisecond
	targetID := "c360.edge.cop.mavlink.asset.system-42"
	requester := &recordingSemLinkGraphRequester{targetID: targetID}

	option, err := semLinkReadbackHandlerOption(cfg, requester, map[string]ownership.OwnerToken{
		copmodel.OwnerCommand: ownership.ExpectedOwnerToken(copmodel.OwnerCommand, "lease-test"),
	})
	if err != nil {
		t.Fatalf("semlink option: %v", err)
	}
	if option == nil {
		t.Fatal("option = nil, want configured handler option")
	}
	handler, err := copapi.NewHandler(copapi.NewFixtureProvider(nil), option)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/cop/semlink/ardupilot/readback",
		strings.NewReader(`{
			"mesh_node_id":"blue-boat",
			"target_asset_id":"c360.edge.cop.mavlink.asset.system-42",
			"vehicle_system_id":42,
			"correlation_id":"corr-42",
			"idempotency_key":"idem-42",
			"ttl_seconds":30
		}`),
	)
	setTrustedSemLinkReadbackHeaders(req)
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if !requester.sawSubject(commandprojector.SubjectGraphQueryEntity) {
		t.Fatalf("graph target query was not issued: %+v", requester.subjects())
	}
	if !requester.sawSubject(commandprojector.SubjectEntityCreateWithTriples) {
		t.Fatalf("graph create was not issued: %+v", requester.subjects())
	}
	if requester.lastCreate.Entity == nil || requester.lastCreate.Entity.ID == "" {
		t.Fatalf("create request missing command-intent entity: %+v", requester.lastCreate)
	}
	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["accepted"] != true ||
		response["native_execution_allowed"] != false ||
		response["companion_transmit_allowed"] != false ||
		response["mutations"].(float64) != 1 ||
		response["authorized_by"] != "operator:semlink-gateway" ||
		response["authorized_mesh_node_id"] != "blue-boat" ||
		response["authority_scope"] != copapi.SemLinkReadbackAuthorityScope {
		t.Fatalf("response = %+v", response)
	}
}

func TestSemLinkReadbackHandlerOptionDisabledByDefault(t *testing.T) {
	option, err := semLinkReadbackHandlerOption(semopsapp.DefaultConfig(), nil, nil)
	if err != nil {
		t.Fatalf("disabled option: %v", err)
	}
	if option != nil {
		t.Fatal("option != nil, want disabled default")
	}
}

func TestSemLinkReadbackHandlerOptionRequiresGraphRequesterWhenEnabled(t *testing.T) {
	cfg := semopsapp.DefaultConfig()
	cfg.COP.SemLinkReadbackEnabled = true
	cfg.COP.OperatorIdentityMode = semopsapp.COPOperatorIdentityModeTrustedHeaders

	_, err := semLinkReadbackHandlerOption(cfg, nil, nil)
	if err == nil || !strings.Contains(err.Error(), semopsapp.EnvCOPSemLinkReadbackEnabled) {
		t.Fatalf("err = %v, want graph requester config error", err)
	}
}

func TestSemLinkReadbackHandlerOptionRequiresCommandOwnerToken(t *testing.T) {
	cfg := semopsapp.DefaultConfig()
	cfg.COP.SemLinkReadbackEnabled = true
	cfg.COP.OperatorIdentityMode = semopsapp.COPOperatorIdentityModeTrustedHeaders

	_, err := semLinkReadbackHandlerOption(cfg, &recordingSemLinkGraphRequester{}, nil)
	if err == nil || !strings.Contains(err.Error(), copmodel.OwnerCommand) {
		t.Fatalf("err = %v, want command owner token error", err)
	}
}

func TestSemLinkReadbackHandlerOptionRequiresTrustedHeaderMode(t *testing.T) {
	cfg := semopsapp.DefaultConfig()
	cfg.COP.SemLinkReadbackEnabled = true

	_, err := semLinkReadbackHandlerOption(
		cfg,
		&recordingSemLinkGraphRequester{},
		map[string]ownership.OwnerToken{
			copmodel.OwnerCommand: ownership.ExpectedOwnerToken(copmodel.OwnerCommand, "lease-test"),
		},
	)
	if err == nil || !strings.Contains(err.Error(), semopsapp.EnvCOPOperatorIdentityMode) {
		t.Fatalf("err = %v, want trusted header mode error", err)
	}
}

type semLinkGraphRequest struct {
	subject string
	data    []byte
	timeout time.Duration
}

type recordingSemLinkGraphRequester struct {
	targetID   string
	requests   []semLinkGraphRequest
	lastCreate graph.CreateEntityWithTriplesRequest
}

func (r *recordingSemLinkGraphRequester) Request(
	_ context.Context,
	subject string,
	data []byte,
	timeout time.Duration,
) ([]byte, error) {
	r.requests = append(r.requests, semLinkGraphRequest{
		subject: subject,
		data:    append([]byte(nil), data...),
		timeout: timeout,
	})
	switch subject {
	case commandprojector.SubjectGraphQueryEntity:
		var query map[string]string
		if err := json.Unmarshal(data, &query); err != nil {
			return nil, err
		}
		if query["id"] != "" && query["id"] == r.targetID {
			return json.Marshal(graph.EntityState{ID: r.targetID})
		}
		return []byte(`{"error":"entity not found"}`), nil
	case commandprojector.SubjectEntityCreateWithTriples:
		if err := json.Unmarshal(data, &r.lastCreate); err != nil {
			return nil, err
		}
		return []byte(`{}`), nil
	case commandprojector.SubjectEntityUpdateWithTriples:
		return []byte(`{}`), nil
	default:
		return []byte(`{}`), nil
	}
}

func (r *recordingSemLinkGraphRequester) sawSubject(subject string) bool {
	for _, request := range r.requests {
		if request.subject == subject {
			return true
		}
	}
	return false
}

func (r *recordingSemLinkGraphRequester) subjects() []string {
	subjects := make([]string, 0, len(r.requests))
	for _, request := range r.requests {
		subjects = append(subjects, request.subject)
	}
	return subjects
}

func setTrustedSemLinkReadbackHeaders(req *http.Request) {
	req.Header.Set(copapi.OperatorAuthenticatedHeader, "true")
	req.Header.Set(copapi.OperatorIDHeader, "operator:semlink-gateway")
	req.Header.Set(copapi.OperatorRoleHeader, copapi.SemLinkReadbackOperatorRole)
	req.Header.Set(copapi.OperatorAuthorityScopeHeader, copapi.SemLinkReadbackAuthorityScope)
	req.Header.Set(copapi.OperatorAuthorityDomainHeader, "boat-blue")
	req.Header.Set(copapi.SemLinkMeshNodeIDHeader, "blue-boat")
}
