from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.create_resource_unit_request_annotations import (
        CreateResourceUnitRequestAnnotations,
    )
    from ..models.create_resource_unit_request_limits import (
        CreateResourceUnitRequestLimits,
    )
    from ..models.create_resource_unit_request_node_selector import (
        CreateResourceUnitRequestNodeSelector,
    )
    from ..models.create_resource_unit_request_requests import (
        CreateResourceUnitRequestRequests,
    )


T = TypeVar("T", bound="CreateResourceUnitRequest")


@_attrs_define
class CreateResourceUnitRequest:
    """
    Example:
        {'description': '2× A100 GPU compute unit.', 'limits': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'},
            'name': 'a100-2x', 'nodeSelector': {'axisml.io/gpu': 'a100'}, 'requests': {'cpu': '16', 'memory': '128Gi',
            'nvidia.com/gpu': '2'}}

    Attributes:
        annotations (CreateResourceUnitRequestAnnotations | Unset): User-defined annotations to set on the unit.
        description (str | Unset): Human-readable description of the unit.
        limits (CreateResourceUnitRequestLimits | Unset): Resource limits per quantity of this unit.
        name (str | Unset): Unit name to create; unique within the pool.
        node_selector (CreateResourceUnitRequestNodeSelector | Unset): Node labels workloads using this unit must match.
        requests (CreateResourceUnitRequestRequests | Unset): Resource requests granted per quantity of this unit.
    """

    annotations: CreateResourceUnitRequestAnnotations | Unset = UNSET
    description: str | Unset = UNSET
    limits: CreateResourceUnitRequestLimits | Unset = UNSET
    name: str | Unset = UNSET
    node_selector: CreateResourceUnitRequestNodeSelector | Unset = UNSET
    requests: CreateResourceUnitRequestRequests | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        description = self.description

        limits: dict[str, Any] | Unset = UNSET
        if not isinstance(self.limits, Unset):
            limits = self.limits.to_dict()

        name = self.name

        node_selector: dict[str, Any] | Unset = UNSET
        if not isinstance(self.node_selector, Unset):
            node_selector = self.node_selector.to_dict()

        requests: dict[str, Any] | Unset = UNSET
        if not isinstance(self.requests, Unset):
            requests = self.requests.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if description is not UNSET:
            field_dict["description"] = description
        if limits is not UNSET:
            field_dict["limits"] = limits
        if name is not UNSET:
            field_dict["name"] = name
        if node_selector is not UNSET:
            field_dict["nodeSelector"] = node_selector
        if requests is not UNSET:
            field_dict["requests"] = requests

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.create_resource_unit_request_annotations import (
            CreateResourceUnitRequestAnnotations,
        )
        from ..models.create_resource_unit_request_limits import (
            CreateResourceUnitRequestLimits,
        )
        from ..models.create_resource_unit_request_node_selector import (
            CreateResourceUnitRequestNodeSelector,
        )
        from ..models.create_resource_unit_request_requests import (
            CreateResourceUnitRequestRequests,
        )

        d = dict(src_dict)
        _annotations = d.pop("annotations", UNSET)
        annotations: CreateResourceUnitRequestAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = CreateResourceUnitRequestAnnotations.from_dict(_annotations)

        description = d.pop("description", UNSET)

        _limits = d.pop("limits", UNSET)
        limits: CreateResourceUnitRequestLimits | Unset
        if isinstance(_limits, Unset):
            limits = UNSET
        else:
            limits = CreateResourceUnitRequestLimits.from_dict(_limits)

        name = d.pop("name", UNSET)

        _node_selector = d.pop("nodeSelector", UNSET)
        node_selector: CreateResourceUnitRequestNodeSelector | Unset
        if isinstance(_node_selector, Unset):
            node_selector = UNSET
        else:
            node_selector = CreateResourceUnitRequestNodeSelector.from_dict(
                _node_selector
            )

        _requests = d.pop("requests", UNSET)
        requests: CreateResourceUnitRequestRequests | Unset
        if isinstance(_requests, Unset):
            requests = UNSET
        else:
            requests = CreateResourceUnitRequestRequests.from_dict(_requests)

        create_resource_unit_request = cls(
            annotations=annotations,
            description=description,
            limits=limits,
            name=name,
            node_selector=node_selector,
            requests=requests,
        )

        create_resource_unit_request.additional_properties = d
        return create_resource_unit_request

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
