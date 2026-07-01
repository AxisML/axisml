from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.resource_unit import ResourceUnit
    from ..models.string_map import StringMap
    from ..models.toleration import Toleration


T = TypeVar("T", bound="ResourcePool")


@_attrs_define
class ResourcePool:
    """
    Example:
        {'annotations': {'axisml.io/cost-center': 'ml-platform', 'axisml.io/created-by': 'admin'}, 'createdAt':
            '2026-06-20T08:00:00Z', 'description': 'A100 GPU resource pool.', 'labels': {'tier': 'gpu'}, 'name': 'gpu-a100',
            'nodeCount': 8, 'nodeSelector': {'axisml.io/gpu': 'a100'}, 'resourceVersion': '184321', 'tolerations':
            [{'effect': 'NoSchedule', 'key': 'nvidia.com/gpu', 'operator': 'Exists'}], 'units': [{'annotations':
            {'axisml.io/cost-center': 'ml-platform', 'axisml.io/created-by': 'admin'}, 'description': '2x A100 GPU compute
            unit.', 'limits': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'}, 'name': 'a100-2x', 'nodeSelector':
            {'arch': 'amd64', 'gpu.product': 'A100'}, 'requests': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'},
            'tolerations': [{'effect': 'NoSchedule', 'key': 'nvidia.com/gpu', 'operator': 'Exists'}]}], 'updatedAt':
            '2026-06-28T09:30:00Z'}

    Attributes:
        created_at (datetime.datetime): Time the pool was created.
        name (str): Cluster-scoped resource pool name (unique across the cluster).
        annotations (StringMap | Unset):
        description (str | Unset): Free-text pool description.
        labels (StringMap | Unset):
        node_count (int | Unset): Number of nodes currently matched by the pool's node selector (read-only).
        node_selector (StringMap | Unset):
        resource_version (str | Unset): Kubernetes resourceVersion for optimistic concurrency.
        tolerations (list[Toleration] | Unset): Pod tolerations applied to workloads scheduled onto the pool.
        units (list[ResourceUnit] | Unset): Resource unit shapes embedded in the pool's spec.units[].
        updated_at (datetime.datetime | Unset): Time the pool was last updated.
    """

    created_at: datetime.datetime
    name: str
    annotations: StringMap | Unset = UNSET
    description: str | Unset = UNSET
    labels: StringMap | Unset = UNSET
    node_count: int | Unset = UNSET
    node_selector: StringMap | Unset = UNSET
    resource_version: str | Unset = UNSET
    tolerations: list[Toleration] | Unset = UNSET
    units: list[ResourceUnit] | Unset = UNSET
    updated_at: datetime.datetime | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        name = self.name

        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        description = self.description

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        node_count = self.node_count

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

        units: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.units, Unset):
            units = []
            for units_item_data in self.units:
                units_item = units_item_data.to_dict()
                units.append(units_item)

        updated_at: str | Unset = UNSET
        if not isinstance(self.updated_at, Unset):
            updated_at = self.updated_at.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "name": name,
            }
        )
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if description is not UNSET:
            field_dict["description"] = description
        if labels is not UNSET:
            field_dict["labels"] = labels
        if node_count is not UNSET:
            field_dict["nodeCount"] = node_count
        if node_selector is not UNSET:
            field_dict["nodeSelector"] = node_selector
        if resource_version is not UNSET:
            field_dict["resourceVersion"] = resource_version
        if tolerations is not UNSET:
            field_dict["tolerations"] = tolerations
        if units is not UNSET:
            field_dict["units"] = units
        if updated_at is not UNSET:
            field_dict["updatedAt"] = updated_at

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.resource_unit import ResourceUnit
        from ..models.string_map import StringMap
        from ..models.toleration import Toleration

        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        name = d.pop("name")

        _annotations = d.pop("annotations", UNSET)
        annotations: StringMap | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = StringMap.from_dict(_annotations)

        description = d.pop("description", UNSET)

        _labels = d.pop("labels", UNSET)
        labels: StringMap | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = StringMap.from_dict(_labels)

        node_count = d.pop("nodeCount", UNSET)

        _node_selector = d.pop("nodeSelector", UNSET)
        node_selector: StringMap | Unset
        if isinstance(_node_selector, Unset):
            node_selector = UNSET
        else:
            node_selector = StringMap.from_dict(_node_selector)

        resource_version = d.pop("resourceVersion", UNSET)

        _tolerations = d.pop("tolerations", UNSET)
        tolerations: list[Toleration] | Unset = UNSET
        if _tolerations is not UNSET:
            tolerations = []
            for tolerations_item_data in _tolerations:
                tolerations_item = Toleration.from_dict(tolerations_item_data)

                tolerations.append(tolerations_item)

        _units = d.pop("units", UNSET)
        units: list[ResourceUnit] | Unset = UNSET
        if _units is not UNSET:
            units = []
            for units_item_data in _units:
                units_item = ResourceUnit.from_dict(units_item_data)

                units.append(units_item)

        _updated_at = d.pop("updatedAt", UNSET)
        updated_at: datetime.datetime | Unset
        if isinstance(_updated_at, Unset):
            updated_at = UNSET
        else:
            updated_at = datetime.datetime.fromisoformat(_updated_at)

        resource_pool = cls(
            created_at=created_at,
            name=name,
            annotations=annotations,
            description=description,
            labels=labels,
            node_count=node_count,
            node_selector=node_selector,
            resource_version=resource_version,
            tolerations=tolerations,
            units=units,
            updated_at=updated_at,
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
