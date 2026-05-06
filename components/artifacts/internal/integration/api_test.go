//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	artmod "github.com/axisml/axisml/components/artifacts/internal/artifact"
	repomod "github.com/axisml/axisml/components/artifacts/internal/repo"
)

// uniqueRepoName returns a per-test repo name so tests don't collide on the
// (tenant, kind, name) unique index when run in the same suite.
func uniqueRepoName(t *testing.T) string {
	return strings.ToLower("repo-" + uuid.New().String()[:8])
}

func TestRepoCreateAndList(t *testing.T) {
	s := setup(t)

	name := uniqueRepoName(t)
	rec := s.drive(t, http.MethodPost, s.tenantPath("/repos"), map[string]any{
		"kind":         "model",
		"name":         name,
		"display_name": "test repo",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create repo: status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = s.drive(t, http.MethodGet, s.tenantPath("/repos"), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list repos: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var listResp struct {
		Items []map[string]any `json:"items"`
		Total int64            `json:"total"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &listResp)
	found := false
	for _, item := range listResp.Items {
		if item["name"] == name {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected repo %q in list, got %+v", name, listResp.Items)
	}
}

func TestRepoCreate_RejectsUnsupportedKind(t *testing.T) {
	s := setup(t)
	rec := s.drive(t, http.MethodPost, s.tenantPath("/repos"), map[string]any{
		"kind": "dataset",
		"name": uniqueRepoName(t),
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported kind, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestArtifact_HappyPath(t *testing.T) {
	s := setup(t)
	repo := uniqueRepoName(t)

	// 1. create repo
	rec := s.drive(t, http.MethodPost, s.tenantPath("/repos"), map[string]any{
		"kind": "model",
		"name": repo,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create repo: status=%d body=%s", rec.Code, rec.Body.String())
	}

	// 2. initiate (POST /artifacts is the collection POST = initiate per
	//    the URL design).
	version := "v1"
	initBody := map[string]any{
		"version": version,
		"spec": map[string]any{
			"framework": "pytorch",
			"format":    "application/vnd.pytorch.bin",
		},
	}
	rec = s.drive(t, http.MethodPost, s.tenantPath("/repos/model/"+repo+"/artifacts"), initBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("initiate: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var initResp artmod.InitiateResult
	if err := json.Unmarshal(rec.Body.Bytes(), &initResp); err != nil {
		t.Fatalf("decode initiate: %v", err)
	}
	if initResp.URI == "" || initResp.UploadCredentials.Username == "" {
		t.Fatalf("initiate result missing fields: %+v", initResp)
	}
	if initResp.StorageKind != "oci" {
		t.Fatalf("expected storage_kind=oci, got %q", initResp.StorageKind)
	}

	// 3. simulate cli `oras push`: fake-zot now has a manifest at the
	//    repoPath:version with a known digest.
	digest := fakeDigest(repo + "@" + version)
	repoPath := "tenants/" + s.tenantName + "/models/" + repo
	s.zot.pushManifest(repoPath, version, digest)

	// 4. complete
	rec = s.drive(t, http.MethodPost,
		s.tenantPath("/repos/model/"+repo+"/artifacts/"+version+"/complete"),
		map[string]any{"digest": digest},
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("complete: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var completeResp artmod.View
	_ = json.Unmarshal(rec.Body.Bytes(), &completeResp)
	if completeResp.Status != artmod.StatusReady {
		t.Fatalf("expected Ready, got %s", completeResp.Status)
	}
	if completeResp.Digest != digest {
		t.Fatalf("digest mismatch: got %s want %s", completeResp.Digest, digest)
	}

	// 5. resolve (inspect)
	rec = s.drive(t, http.MethodGet,
		s.tenantPath("/repos/model/"+repo+"/artifacts/"+version+"/resolve?usage=inspect"),
		nil,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("resolve inspect: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var inspectResp artmod.ResolveResult
	_ = json.Unmarshal(rec.Body.Bytes(), &inspectResp)
	if inspectResp.PullCredentials != nil {
		t.Fatalf("inspect must not return pull_credentials in MVP, got %+v", inspectResp.PullCredentials)
	}
	if inspectResp.Digest != digest {
		t.Fatalf("inspect digest = %s, want %s", inspectResp.Digest, digest)
	}

	// 6. resolve (download)
	rec = s.drive(t, http.MethodGet,
		s.tenantPath("/repos/model/"+repo+"/artifacts/"+version+"/resolve?usage=download"),
		nil,
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("resolve download: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var downloadResp artmod.ResolveResult
	_ = json.Unmarshal(rec.Body.Bytes(), &downloadResp)
	if downloadResp.PullCredentials == nil {
		t.Fatalf("download must return pull_credentials, got nil")
	}

	// 7. delete artifact + GC tick → Deleted
	rec = s.drive(t, http.MethodDelete, s.tenantPath("/repos/model/"+repo+"/artifacts/"+version), nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("delete: status=%d body=%s", rec.Code, rec.Body.String())
	}
	s.gcW.Tick(context.Background())

	// Direct DB read to confirm Deleted; GET returns the row regardless of
	// status until soft-delete sets deleted_at.
	rec = s.drive(t, http.MethodGet,
		s.tenantPath("/repos/model/"+repo+"/artifacts/"+version), nil,
	)
	if rec.Code == http.StatusOK {
		var view artmod.View
		_ = json.Unmarshal(rec.Body.Bytes(), &view)
		if view.Status != artmod.StatusDeleted {
			t.Fatalf("expected Deleted, got %s", view.Status)
		}
	}
}

func TestArtifact_DigestMismatch(t *testing.T) {
	s := setup(t)
	repo := uniqueRepoName(t)
	rec := s.drive(t, http.MethodPost, s.tenantPath("/repos"), map[string]any{"kind": "model", "name": repo})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create repo: %d", rec.Code)
	}

	// initiate
	rec = s.drive(t, http.MethodPost, s.tenantPath("/repos/model/"+repo+"/artifacts"), map[string]any{
		"version": "v1",
		"spec":    map[string]any{"framework": "pytorch", "format": "application/x-bin"},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("initiate: %d %s", rec.Code, rec.Body.String())
	}

	// fake-zot has a different digest than the cli will claim
	repoPath := "tenants/" + s.tenantName + "/models/" + repo
	s.zot.pushManifest(repoPath, "v1", fakeDigest("real"))

	rec = s.drive(t, http.MethodPost, s.tenantPath("/repos/model/"+repo+"/artifacts/v1/complete"),
		map[string]any{"digest": fakeDigest("wrong")},
	)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 on digest mismatch, got %d body=%s", rec.Code, rec.Body.String())
	}

	// state should be Failed now
	rec = s.drive(t, http.MethodGet, s.tenantPath("/repos/model/"+repo+"/artifacts/v1"), nil)
	var view artmod.View
	_ = json.Unmarshal(rec.Body.Bytes(), &view)
	if view.Status != artmod.StatusFailed {
		t.Fatalf("expected Failed, got %s", view.Status)
	}
}

func TestGC_UploadingTTLExpiry(t *testing.T) {
	s := setup(t)
	repo := uniqueRepoName(t)
	rec := s.drive(t, http.MethodPost, s.tenantPath("/repos"), map[string]any{"kind": "model", "name": repo})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create repo: %d", rec.Code)
	}

	// initiate but never complete
	rec = s.drive(t, http.MethodPost, s.tenantPath("/repos/model/"+repo+"/artifacts"), map[string]any{
		"version": "v1",
		"spec":    map[string]any{"framework": "pytorch", "format": "application/x-bin"},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("initiate: %d", rec.Code)
	}

	// Forcibly age the row past the TTL by writing created_at directly.
	if err := s.gormDB.Exec(`
		UPDATE artifacts
		   SET created_at = now() - interval '25 hour'
		 WHERE repo_id = (SELECT id FROM artifact_repos WHERE name = ? AND deleted_at IS NULL)
	`, repo).Error; err != nil {
		t.Fatalf("age artifact: %v", err)
	}

	s.gcW.Tick(context.Background())

	rec = s.drive(t, http.MethodGet, s.tenantPath("/repos/model/"+repo+"/artifacts/v1"), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get: %d %s", rec.Code, rec.Body.String())
	}
	var view artmod.View
	_ = json.Unmarshal(rec.Body.Bytes(), &view)
	if view.Status != artmod.StatusFailed {
		t.Fatalf("expected Failed after TTL, got %s", view.Status)
	}
}

func TestRepoDeleteCascade(t *testing.T) {
	s := setup(t)
	repo := uniqueRepoName(t)
	rec := s.drive(t, http.MethodPost, s.tenantPath("/repos"), map[string]any{"kind": "model", "name": repo})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create repo: %d", rec.Code)
	}
	// initiate one artifact (Uploading state) — DELETE should still cascade
	rec = s.drive(t, http.MethodPost, s.tenantPath("/repos/model/"+repo+"/artifacts"), map[string]any{
		"version": "v1",
		"spec":    map[string]any{"framework": "pytorch", "format": "application/x-bin"},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("initiate: %d", rec.Code)
	}

	// DELETE the repo
	rec = s.drive(t, http.MethodDelete, s.tenantPath("/repos/model/"+repo), nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("delete repo: %d", rec.Code)
	}

	// GC ticks until repo finalizes (cap iterations to keep tests bounded).
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		s.gcW.Tick(context.Background())
		var status string
		if err := s.gormDB.Raw(`SELECT status FROM artifact_repos WHERE name = ?`, repo).
			Scan(&status).Error; err != nil {
			t.Fatalf("read repo status: %v", err)
		}
		if status == repomod.StatusDeleted {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("repo did not reach Deleted within deadline")
}
