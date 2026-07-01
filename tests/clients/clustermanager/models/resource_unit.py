from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.resource_unit_annotations import ResourceUnitAnnotations
    from ..models.resource_unit_limits import ResourceUnitLimits
    from ..models.resource_unit_node_selector import ResourceUnitNodeSelector
    from ..models.resource_unit_requests import ResourceUnitRequests


T = TypeVar("T", bound="ResourceUnit")


@_attrs_define
class ResourceUnit:
    """
    Example:
        {'annotations': {'tenant.axisml.io/managed-by': 'platform'}, 'description': '2× A100 GPU compute unit.',
            'limits': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'}, 'name': 'a100-2x', 'nodeSelector':
            {'axisml.io/gpu': 'a100'}, 'requests': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'}}

    Attributes:
        limits (ResourceUnitLimits): Resource limits per quantity of this unit.
        name (str): Unit name; unique within the pool and immutable.
        requests (ResourceUnitRequests): Resource requests granted per quantity of this unit (e.g. cpu, memory,
            nvidia.com/gpu).
        annotations (ResourceUnitAnnotations | Unset): User-defined annotations on the unit.
        description (str | Unset): Human-readable description of the unit.
        node_selector (ResourceUnitNodeSelector | Unset): Node labels workloads using this unit must match (overrides
            the pool selector).
    """

    limits: ResourceUnitLimits
    name: str
    requests: ResourceUnitRequests
    annotations: ResourceUnitAnnotations | Unset = UNSET
    description: str | Unset = UNSET
    node_selector: ResourceUnitNodeSelector | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        limits = self.limits.to_dict()

        name = self.name

        requests = self.requests.to_dict()

        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        description = self.description

        node_selector: dict[str, Any] | Unset = UNSET
        if not isinstance(self.node_selector, Unset):
            node_selector = self.node_selector.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "limits": limits,
                "name": name,
                "requests": requests,
            }
        )
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if description is not UNSET:
            field_dict["description"] = description
        if node_selector is not UNSET:
            field_dict["nodeSelector"] = node_selector

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.resource_unit_annotations import ResourceUnitAnnotations
        from ..models.resource_unit_limits import ResourceUnitLimits
        from ..models.resource_unit_node_selector import ResourceUnitNodeSelector
        from ..models.resource_unit_requests import ResourceUnitRequests

        d = dict(src_dict)
        limits = ResourceUnitLimits.from_dict(d.pop("limits"))

        name = d.pop("name")

        requests = ResourceUnitRequests.from_dict(d.pop("requests"))

        _annotations = d.pop("annotations", UNSET)
        annotations: ResourceUnitAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = ResourceUnitAnnotations.from_dict(_annotations)

        description = d.pop("description", UNSET)

        _node_selector = d.pop("nodeSelector", UNSET)
        node_selector: ResourceUnitNodeSelector | Unset
        if isinstance(_node_selector, Unset):
            node_selector = UNSET
        else:
            node_selector = ResourceUnitNodeSelector.from_dict(_node_selector)

        resource_unit = cls(
            limits=limits,
            name=name,
            requests=requests,
            annotations=annotations,
            description=description,
            node_selector=node_selector,
        )

        resource_unit.additional_properties = d
        return resource_unit

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
