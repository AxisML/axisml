from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.resource_map import ResourceMap
    from ..models.resource_unit_create_request import ResourceUnitCreateRequest
    from ..models.string_map import StringMap


T = TypeVar("T", bound="ResourcePoolCreateRequest")


@_attrs_define
class ResourcePoolCreateRequest:
    """
    Example:
        {'capacity': {'cpu': '64', 'memory': '512Gi', 'nvidia.com/gpu': '8'}, 'description': 'A100 GPU resource pool.',
            'labels': {'tier': 'gpu'}, 'name': 'gpu-a100', 'nodeSelector': {'axisml.io/gpu': 'a100'}, 'units':
            [{'description': '2x A100 GPU compute unit.', 'limits': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'},
            'name': 'a100-2x', 'requests': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'}}]}

    Attributes:
        name (str): Cluster-scoped resource pool name (unique across the cluster).
        annotations (StringMap | Unset):
        capacity (ResourceMap | Unset): Kubernetes-style resource quantity map (e.g., {"cpu": "100", "memory": "1Ti",
            "nvidia.com/gpu": "8"}).
        description (str | Unset): Free-text pool description.
        labels (StringMap | Unset):
        node_selector (StringMap | Unset):
        units (list[ResourceUnitCreateRequest] | Unset): Resource unit shapes to embed in the pool.
    """

    name: str
    annotations: StringMap | Unset = UNSET
    capacity: ResourceMap | Unset = UNSET
    description: str | Unset = UNSET
    labels: StringMap | Unset = UNSET
    node_selector: StringMap | Unset = UNSET
    units: list[ResourceUnitCreateRequest] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        capacity: dict[str, Any] | Unset = UNSET
        if not isinstance(self.capacity, Unset):
            capacity = self.capacity.to_dict()

        description = self.description

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        node_selector: dict[str, Any] | Unset = UNSET
        if not isinstance(self.node_selector, Unset):
            node_selector = self.node_selector.to_dict()

        units: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.units, Unset):
            units = []
            for units_item_data in self.units:
                units_item = units_item_data.to_dict()
                units.append(units_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
            }
        )
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if capacity is not UNSET:
            field_dict["capacity"] = capacity
        if description is not UNSET:
            field_dict["description"] = description
        if labels is not UNSET:
            field_dict["labels"] = labels
        if node_selector is not UNSET:
            field_dict["nodeSelector"] = node_selector
        if units is not UNSET:
            field_dict["units"] = units

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.resource_map import ResourceMap
        from ..models.resource_unit_create_request import ResourceUnitCreateRequest
        from ..models.string_map import StringMap

        d = dict(src_dict)
        name = d.pop("name")

        _annotations = d.pop("annotations", UNSET)
        annotations: StringMap | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = StringMap.from_dict(_annotations)

        _capacity = d.pop("capacity", UNSET)
        capacity: ResourceMap | Unset
        if isinstance(_capacity, Unset):
            capacity = UNSET
        else:
            capacity = ResourceMap.from_dict(_capacity)

        description = d.pop("description", UNSET)

        _labels = d.pop("labels", UNSET)
        labels: StringMap | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = StringMap.from_dict(_labels)

        _node_selector = d.pop("nodeSelector", UNSET)
        node_selector: StringMap | Unset
        if isinstance(_node_selector, Unset):
            node_selector = UNSET
        else:
            node_selector = StringMap.from_dict(_node_selector)

        _units = d.pop("units", UNSET)
        units: list[ResourceUnitCreateRequest] | Unset = UNSET
        if _units is not UNSET:
            units = []
            for units_item_data in _units:
                units_item = ResourceUnitCreateRequest.from_dict(units_item_data)

                units.append(units_item)

        resource_pool_create_request = cls(
            name=name,
            annotations=annotations,
            capacity=capacity,
            description=description,
            labels=labels,
            node_selector=node_selector,
            units=units,
        )

        resource_pool_create_request.additional_properties = d
        return resource_pool_create_request

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
