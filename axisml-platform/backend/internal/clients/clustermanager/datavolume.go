package clustermanager

import (
	"context"
	"encoding/json"

	"github.com/axisml/axisml/axisml-platform/backend/internal/clients/clienterr"
	gen "github.com/axisml/axisml/axisml-platform/backend/internal/clients/clustermanager/generated"
)

// Clean-named aliases for the generated Volume wire types.
type (
	Volume       = gen.Volume
	VolumeStatus = gen.ServerVolumeStatus
	VolumeMount  = gen.ServerVolumeMount
	VolumeCreate = gen.CreateVolumeRequest
	VolumePatch  = gen.PatchVolumeRequest
	StorageClass = gen.ServerStorageClass
)

// ListStorageClasses returns the cluster's storage classes for the new-volume
// picker.
func (c *Client) ListStorageClasses(ctx context.Context) ([]StorageClass, error) {
	res, err := c.gen.ListStorageClassesWithResponse(ctx)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 == nil {
		return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
	}
	return res.JSON200.Items, nil
}

// ListVolumes returns the managed data volumes in a physical namespace. Items
// are normalised to the bare Volume shape.
func (c *Client) ListVolumes(ctx context.Context, namespace, labelSelector string) ([]Volume, error) {
	params := &gen.ListVolumesParams{}
	if namespace != "" {
		params.Namespace = &namespace
	}
	if labelSelector != "" {
		params.LabelSelector = &labelSelector
	}
	res, err := c.gen.ListVolumesWithResponse(ctx, params)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 == nil {
		return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
	}
	out := make([]Volume, 0, len(res.JSON200.Items))
	for i := range res.JSON200.Items {
		var v Volume
		b, _ := json.Marshal(res.JSON200.Items[i])
		if err := json.Unmarshal(b, &v); err != nil {
			return nil, clienterr.Transport(service, err)
		}
		out = append(out, v)
	}
	return out, nil
}

// GetVolume returns one volume with live status and mount occupancy.
func (c *Client) GetVolume(ctx context.Context, namespace, name string) (*Volume, error) {
	res, err := c.gen.GetVolumeWithResponse(ctx, namespace, name)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 != nil {
		return res.JSON200, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// CreateVolume materialises a data volume (idempotent).
func (c *Client) CreateVolume(ctx context.Context, body VolumeCreate) (*Volume, error) {
	res, err := c.gen.CreateVolumeWithResponse(ctx, body)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON201 != nil {
		return res.JSON201, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// UpdateVolume expands and/or relabels a data volume.
func (c *Client) UpdateVolume(ctx context.Context, namespace, name string, body VolumePatch) (*Volume, error) {
	res, err := c.gen.UpdateVolumeWithResponse(ctx, namespace, name, body)
	if err != nil {
		return nil, clienterr.Transport(service, err)
	}
	if res.JSON200 != nil {
		return res.JSON200, nil
	}
	return nil, clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}

// DeleteVolume reclaims a data volume. Occupancy-guarded downstream unless force.
func (c *Client) DeleteVolume(ctx context.Context, namespace, name string, force bool) error {
	params := &gen.DeleteVolumeParams{}
	if force {
		params.Force = &force
	}
	res, err := c.gen.DeleteVolumeWithResponse(ctx, namespace, name, params)
	if err != nil {
		return clienterr.Transport(service, err)
	}
	if res.HTTPResponse != nil && res.HTTPResponse.StatusCode < 300 {
		return nil
	}
	return clienterr.FromResponse(service, res.HTTPResponse, res.Body)
}
