// Package artifacthub is a thin, typed wrapper over the generated artifact-hub
// client. The {namespace} path segment is the tenant scope. The generated client
// exposes per-kind methods; this wrapper dispatches on a kind string ("model" /
// "image") so the Models/Images modules share one path.
//
// NOTE: the current artifact-hub initiate input has no `source` field and no
// external (no-upload) registration path; those require the artifact-hub
// push-down (#4) before Platform can pass source / register external versions.
package artifacthub

import (
	"context"
	"net/http"
	"time"

	gen "github.com/axisml/axisml/axisml-platform/backend/internal/clients/artifacthub/generated"
	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/clienterr"
	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/reqedit"
	apperrors "github.com/axisml/axisml/axisml-platform/backend/pkg/errors"
)

const service = "artifacts"

// Re-exported generated types.
type (
	Artifact         = gen.Artifact
	InitiateRequest  = gen.ArtifactInitiateRequest
	InitiateResponse = gen.ArtifactInitiateResponse
	ResolveResponse  = gen.ArtifactResolveResponse
	PatchRequest     = gen.ArtifactPatchRequest
)

// Client wraps the generated artifact-hub client.
type Client struct{ gen *gen.ClientWithResponses }

// New builds an artifact-hub client for baseURL.
func New(baseURL string, timeout time.Duration) (*Client, error) {
	c, err := gen.NewClientWithResponses(baseURL,
		gen.WithHTTPClient(&http.Client{Timeout: timeout}),
		gen.WithRequestEditorFn(reqedit.Identity),
	)
	if err != nil {
		return nil, err
	}
	return &Client{gen: c}, nil
}

func badKind(kind string) error {
	return apperrors.Newf(apperrors.ClassValidation, "unknown artifact kind %q", kind).WithReason("invalid-kind")
}

// Initiate starts a new version upload.
func (c *Client) Initiate(ctx context.Context, ns, kind, name string, in InitiateRequest) (*InitiateResponse, error) {
	switch kind {
	case "model":
		res, err := c.gen.InitiateModelWithResponse(ctx, ns, name, in)
		if err != nil {
			return nil, clienterr.Transport(service, err)
		}
		if res.JSON201 != nil {
			return res.JSON201, nil
		}
		return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
	case "image":
		res, err := c.gen.InitiateImageWithResponse(ctx, ns, name, in)
		if err != nil {
			return nil, clienterr.Transport(service, err)
		}
		if res.JSON201 != nil {
			return res.JSON201, nil
		}
		return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
	}
	return nil, badKind(kind)
}

// Get returns one artifact version.
func (c *Client) Get(ctx context.Context, ns, kind, name, version string) (*Artifact, error) {
	switch kind {
	case "model":
		res, err := c.gen.GetModelWithResponse(ctx, ns, name, version)
		return view(res.JSON200, res.HTTPResponse, res.Body, err)
	case "image":
		res, err := c.gen.GetImageWithResponse(ctx, ns, name, version)
		return view(res.JSON200, res.HTTPResponse, res.Body, err)
	}
	return nil, badKind(kind)
}

// Update patches mutable version metadata.
func (c *Client) Update(ctx context.Context, ns, kind, name, version string, in PatchRequest) (*Artifact, error) {
	switch kind {
	case "model":
		res, err := c.gen.UpdateModelWithResponse(ctx, ns, name, version, in)
		return view(res.JSON200, res.HTTPResponse, res.Body, err)
	case "image":
		res, err := c.gen.UpdateImageWithResponse(ctx, ns, name, version, in)
		return view(res.JSON200, res.HTTPResponse, res.Body, err)
	}
	return nil, badKind(kind)
}

// Delete soft-deletes one artifact version.
func (c *Client) Delete(ctx context.Context, ns, kind, name, version string) error {
	switch kind {
	case "model":
		res, err := c.gen.DeleteModelWithResponse(ctx, ns, name, version)
		return ok(res.HTTPResponse, res.Body, err)
	case "image":
		res, err := c.gen.DeleteImageWithResponse(ctx, ns, name, version)
		return ok(res.HTTPResponse, res.Body, err)
	}
	return badKind(kind)
}

// Complete finalizes a version upload with its digest.
func (c *Client) Complete(ctx context.Context, ns, kind, name, version, digest string) (*Artifact, error) {
	body := gen.ArtifactCompleteRequest{Digest: digest}
	switch kind {
	case "model":
		res, err := c.gen.CompleteModelWithResponse(ctx, ns, name, version, body)
		return view(res.JSON200, res.HTTPResponse, res.Body, err)
	case "image":
		res, err := c.gen.CompleteImageWithResponse(ctx, ns, name, version, body)
		return view(res.JSON200, res.HTTPResponse, res.Body, err)
	}
	return nil, badKind(kind)
}

// Resolve returns download credentials / inspect URI for a version.
func (c *Client) Resolve(ctx context.Context, ns, kind, name, version, usage string) (*ResolveResponse, error) {
	switch kind {
	case "model":
		res, err := c.gen.ResolveModelWithResponse(ctx, ns, name, version, &gen.ResolveModelParams{Usage: &usage})
		if err != nil {
			return nil, clienterr.Transport(service, err)
		}
		if res.JSON200 != nil {
			return res.JSON200, nil
		}
		return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
	case "image":
		res, err := c.gen.ResolveImageWithResponse(ctx, ns, name, version, &gen.ResolveImageParams{Usage: &usage})
		if err != nil {
			return nil, clienterr.Transport(service, err)
		}
		if res.JSON200 != nil {
			return res.JSON200, nil
		}
		return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
	}
	return nil, badKind(kind)
}

// ListVersions lists a definition's versions.
func (c *Client) ListVersions(ctx context.Context, ns, kind, name string) ([]Artifact, error) {
	switch kind {
	case "model":
		res, err := c.gen.ListModelVersionsWithResponse(ctx, ns, name, &gen.ListModelVersionsParams{})
		if err != nil {
			return nil, clienterr.Transport(service, err)
		}
		if res.JSON200 == nil {
			return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
		}
		return res.JSON200.Items, nil
	case "image":
		res, err := c.gen.ListImageVersionsWithResponse(ctx, ns, name, &gen.ListImageVersionsParams{})
		if err != nil {
			return nil, clienterr.Transport(service, err)
		}
		if res.JSON200 == nil {
			return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
		}
		return res.JSON200.Items, nil
	}
	return nil, badKind(kind)
}

func view(j *Artifact, resp *http.Response, body []byte, err error) (*Artifact, error) {
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if j != nil {
		return j, nil
	}
	return nil, clienterr.FromResponse(service, resp, body)
}

func ok(resp *http.Response, body []byte, err error) error {
	if err != nil {
		return clienterr.Transport(service, err)
	}
	if resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return clienterr.FromResponse(service, resp, body)
}
