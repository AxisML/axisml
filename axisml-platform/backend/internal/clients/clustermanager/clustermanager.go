// Package clustermanager is a thin, typed wrapper over the generated
// cluster-manager client (internal/clients/clustermanager/generated). It injects
// the X-Axisml-User identity header and maps downstream problems to Platform
// business errors. The generated code is the source of truth for the wire types
// (regenerate with `make client-gen`); this file only adds ergonomics.
package clustermanager

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/clienterr"
	gen "github.com/axisml/axisml/axisml-platform/backend/internal/clients/clustermanager/generated"
	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/reqedit"
)

const service = "cluster-manager"

// Clean-named aliases for the generated wire types, so callers never handle the
// spec-derived, package-prefixed names. The generated package remains the source of truth.
type (
	Tenant         = gen.Tenant
	Quota          = gen.ServerQuota
	QuotaUnit      = gen.ServerQuotaUnit
	QuotaResources = gen.ServerQuotaResources
	QuotaStatus    = gen.ServerQuotaStatus
	Namespace      = gen.Tenantv1alpha1NamespaceSpec
	InitResources  = gen.Tenantv1alpha1InitResources
)

// Client wraps the generated cluster-manager client.
type Client struct{ gen *gen.ClientWithResponses }

// New builds a cluster-manager client for baseURL.
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

// TenantVolume is a predefined data volume seeded into the tenant namespace on
// provisioning (written to the Tenant CR's initResources.volumes[]).
type TenantVolume struct {
	Name         string
	Size         string
	StorageClass string
	AccessModes  []string
	Description  string
}

// CreateTenantInput is the create payload (name + namespace + quotas + meta +
// predefined volumes).
type CreateTenantInput struct {
	Name          string
	NamespaceName string
	DisplayName   string
	Labels        map[string]string
	Annotations   map[string]string
	Quotas        []Quota
	Volumes       []TenantVolume
}

// CreateTenant materialises a Tenant CR.
func (c *Client) CreateTenant(ctx context.Context, in CreateTenantInput) (*Tenant, error) {
	body := gen.CreateTenantRequest{
		Name:      strPtr(in.Name),
		Namespace: &Namespace{Name: in.NamespaceName},
	}
	if len(in.Labels) > 0 {
		body.Labels = &in.Labels
	}
	if len(in.Annotations) > 0 {
		body.Annotations = &in.Annotations
	}
	if len(in.Quotas) > 0 {
		body.Quotas = &in.Quotas
	}
	if len(in.Volumes) > 0 {
		body.InitResources = &gen.Tenantv1alpha1InitResources{Volumes: toGenVolumes(in.Volumes)}
	}
	res, err := c.gen.CreateTenantWithResponse(ctx, body)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON201 != nil {
		return res.JSON201, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// GetTenant returns a tenant with live status.
func (c *Client) GetTenant(ctx context.Context, name string) (*Tenant, error) {
	res, err := c.gen.GetTenantWithResponse(ctx, name)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 != nil {
		return res.JSON200, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// UpdateTenant patches tenant metadata (labels/annotations sync).
func (c *Client) UpdateTenant(ctx context.Context, name string, labels, annotations map[string]string) (*Tenant, error) {
	body := gen.PatchTenantRequest{}
	if labels != nil {
		body.Labels = &labels
	}
	if annotations != nil {
		body.Annotations = &annotations
	}
	res, err := c.gen.UpdateTenantWithResponse(ctx, name, body)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 != nil {
		return res.JSON200, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// DeleteTenant removes a Tenant CR (idempotent).
func (c *Client) DeleteTenant(ctx context.Context, name string) error {
	res, err := c.gen.DeleteTenantWithResponse(ctx, name)
	if err != nil {
		return clienterr.Transport(service, err)
	}
	if res.HTTPResponse != nil && res.HTTPResponse.StatusCode < 300 {
		return nil
	}
	return clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// ListTenants lists tenants, optionally filtered by labelSelector. Items are
// normalised to the bare Tenant shape.
func (c *Client) ListTenants(ctx context.Context, labelSelector string) ([]Tenant, error) {
	params := &gen.ListTenantsParams{}
	if labelSelector != "" {
		params.LabelSelector = &labelSelector
	}
	res, err := c.gen.ListTenantsWithResponse(ctx, params)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 == nil {
		return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
	}
	out := make([]Tenant, 0, len(res.JSON200.Items))
	for i := range res.JSON200.Items {
		var t Tenant
		b, _ := json.Marshal(res.JSON200.Items[i])
		if err := json.Unmarshal(b, &t); err != nil {
			return nil, clienterr.Transport(service, err)
		}
		out = append(out, t)
	}
	return out, nil
}

// SetQuota creates or replaces a tenant's quota for one pool, in either the
// units business form or direct min/max mode (mutually exclusive).
func (c *Client) SetQuota(ctx context.Context, tenant, pool string, units []QuotaUnit, direct *QuotaResources) error {
	body := gen.SetQuotaRequest{Pool: strPtr(pool)}
	if direct != nil {
		body.Quota = direct
	} else {
		body.Units = &units
	}
	res, err := c.gen.SetTenantQuotaWithResponse(ctx, tenant, body)
	if err != nil {
		return clienterr.Transport(service, err)
	}
	if res.JSON200 != nil {
		return nil
	}
	return clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// UpdateQuota replaces an existing pool quota's input, in either the units
// business form or direct min/max mode (mutually exclusive).
func (c *Client) UpdateQuota(ctx context.Context, tenant, pool string, units []QuotaUnit, direct *QuotaResources) error {
	body := gen.PatchQuotaRequest{}
	if direct != nil {
		body.Quota = direct
	} else {
		body.Units = &units
	}
	res, err := c.gen.UpdateTenantQuotaWithResponse(ctx, tenant, pool, body)
	if err != nil {
		return clienterr.Transport(service, err)
	}
	if res.JSON200 != nil {
		return nil
	}
	return clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// DeleteQuota removes a tenant's pool quota (idempotent).
func (c *Client) DeleteQuota(ctx context.Context, tenant, pool string) error {
	res, err := c.gen.DeleteTenantQuotaWithResponse(ctx, tenant, pool)
	if err != nil {
		return clienterr.Transport(service, err)
	}
	if res.HTTPResponse != nil && res.HTTPResponse.StatusCode < 300 {
		return nil
	}
	return clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// ListQuotas lists a tenant's per-pool quotas.
func (c *Client) ListQuotas(ctx context.Context, tenant string) ([]Quota, error) {
	res, err := c.gen.ListTenantQuotasWithResponse(ctx, tenant)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 == nil {
		return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
	}
	return res.JSON200.Items, nil
}

func strPtr(s string) *string { return &s }

// toGenVolumes maps the client's TenantVolume inputs to the generated CR
// initResources.volumes shape.
func toGenVolumes(in []TenantVolume) *[]gen.Tenantv1alpha1VolumeSpec {
	out := make([]gen.Tenantv1alpha1VolumeSpec, 0, len(in))
	for _, v := range in {
		gv := gen.Tenantv1alpha1VolumeSpec{Name: v.Name}
		if v.Size != "" {
			gv.Size = strPtr(v.Size)
		}
		if v.StorageClass != "" {
			gv.StorageClass = strPtr(v.StorageClass)
		}
		if v.Description != "" {
			gv.Description = strPtr(v.Description)
		}
		if len(v.AccessModes) > 0 {
			modes := append([]string(nil), v.AccessModes...)
			gv.AccessModes = &modes
		}
		out = append(out, gv)
	}
	return &out
}
