from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_toleration import Corev1Toleration
    from ..models.resource_pool_annotations import ResourcePoolAnnotations
    from ..models.resource_pool_labels import ResourcePoolLabels
    from ..models.resource_pool_node_selector import ResourcePoolNodeSelector
    from ..models.server_resource_unit import ServerResourceUnit


T = TypeVar("T", bound="ResourcePool")


@_attrs_define
class ResourcePool:
    """
    Example:
        {'annotations': {'tenant.axisml.io/managed-by': 'platform'}, 'createdAt': '2026-06-20T08:00:00Z', 'description':
            'A100 GPU resource pool.', 'labels': {'tier': 'gpu'}, 'name': 'gpu-a100', 'nodeSelector': {'axisml.io/gpu':
            'a100'}, 'resourceVersion': '184729', 'tolerations': [{'effect': 'NoSchedule', 'key': 'nvidia.com/gpu',
            'operator': 'Exists'}], 'units': [{'annotations': {'tenant.axisml.io/managed-by': 'platform'}, 'description':
            '2× A100 GPU compute unit.', 'limits': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'}, 'name':
            'a100-2x', 'nodeSelector': {'axisml.io/gpu': 'a100'}, 'requests': {'cpu': '16', 'memory': '128Gi',
            'nvidia.com/gpu': '2'}}]}

    Attributes:
        created_at (datetime.datetime): Pool creation timestamp (RFC3339).
        name (str): Pool name; the stable, immutable handle (CR metadata.name).
        units (list[ServerResourceUnit]): Resource units (allocatable shapes) offered by this pool.
        annotations (ResourcePoolAnnotations | Unset): User-defined annotations on the pool.
        description (str | Unset): Human-readable description of the pool.
        labels (ResourcePoolLabels | Unset): User-defined labels on the pool.
        node_selector (ResourcePoolNodeSelector | Unset): Node labels that workloads scheduled into this pool must
            match.
        resource_version (str | Unset): Opaque CR resourceVersion for optimistic concurrency.
        tolerations (list[Corev1Toleration] | Unset): Tolerations applied to workloads scheduled into this pool.
    """

    created_at: datetime.datetime
    name: str
    units: list[ServerResourceUnit]
    annotations: ResourcePoolAnnotations | Unset = UNSET
    description: str | Unset = UNSET
    labels: ResourcePoolLabels | Unset = UNSET
    node_selector: ResourcePoolNodeSelector | Unset = UNSET
    resource_version: str | Unset = UNSET
    tolerations: list[Corev1Toleration] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        name = self.name

        units = []
        for units_item_data in self.units:
            units_item = units_item_data.to_dict()
            units.append(units_item)

        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        description = self.description

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        node_selector: dict[str, Any] | Unset = UNSET
        if not isinstance(self.node_selector, Unset):
            node_selector = self.node_selector.to_dict()

        resource_version = self.resource_version

        tolerations: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.tolerations, Unset):
            tolerations = []
            for tolerations_item_data in self.tolerations:
                tolerations_item = tolerations_item_data.to_dict()
                tolerations.append(tolerations_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "name": name,
                "units": units,
            }
        )
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if description is not UNSET:
            field_dict["description"] = description
        if labels is not UNSET:
            field_dict["labels"] = labels
        if node_selector is not UNSET:
            field_dict["nodeSelector"] = node_selector
        if resource_version is not UNSET:
            field_dict["resourceVersion"] = resource_version
        if tolerations is not UNSET:
            field_dict["tolerations"] = tolerations

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_toleration import Corev1Toleration
        from ..models.resource_pool_annotations import ResourcePoolAnnotations
        from ..models.resource_pool_labels import ResourcePoolLabels
        from ..models.resource_pool_node_selector import ResourcePoolNodeSelector
        from ..models.server_resource_unit import ServerResourceUnit

        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        name = d.pop("name")

        units = []
        _units = d.pop("units")
        for units_item_data in _units:
            units_item = ServerResourceUnit.from_dict(units_item_data)

            units.append(units_item)

        _annotations = d.pop("annotations", UNSET)
        annotations: ResourcePoolAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = ResourcePoolAnnotations.from_dict(_annotations)

        description = d.pop("description", UNSET)

        _labels = d.pop("labels", UNSET)
        labels: ResourcePoolLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = ResourcePoolLabels.from_dict(_labels)

        _node_selector = d.pop("nodeSelector", UNSET)
        node_selector: ResourcePoolNodeSelector | Unset
        if isinstance(_node_selector, Unset):
            node_selector = UNSET
        else:
            node_selector = ResourcePoolNodeSelector.from_dict(_node_selector)

        resource_version = d.pop("resourceVersion", UNSET)

        _tolerations = d.pop("tolerations", UNSET)
        tolerations: list[Corev1Toleration] | Unset = UNSET
        if _tolerations is not UNSET:
            tolerations = []
            for tolerations_item_data in _tolerations:
                tolerations_item = Corev1Toleration.from_dict(tolerations_item_data)

                tolerations.append(tolerations_item)

        resource_pool = cls(
            created_at=created_at,
            name=name,
            units=units,
            annotations=annotations,
            description=description,
            labels=labels,
            node_selector=node_selector,
            resource_version=resource_version,
            tolerations=tolerations,
        )

        resource_pool.additional_properties = d
        return resource_pool

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
