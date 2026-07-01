from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.server_create_resource_unit_request_annotations import (
        ServerCreateResourceUnitRequestAnnotations,
    )
    from ..models.server_create_resource_unit_request_limits import (
        ServerCreateResourceUnitRequestLimits,
    )
    from ..models.server_create_resource_unit_request_node_selector import (
        ServerCreateResourceUnitRequestNodeSelector,
    )
    from ..models.server_create_resource_unit_request_requests import (
        ServerCreateResourceUnitRequestRequests,
    )


T = TypeVar("T", bound="ServerCreateResourceUnitRequest")


@_attrs_define
class ServerCreateResourceUnitRequest:
    """
    Attributes:
        limits (ServerCreateResourceUnitRequestLimits): Resource limits per quantity of this unit.
        name (str): Unit name to create; unique within the pool.
        requests (ServerCreateResourceUnitRequestRequests): Resource requests granted per quantity of this unit.
        annotations (ServerCreateResourceUnitRequestAnnotations | Unset): User-defined annotations to set on the unit.
        description (str | Unset): Human-readable description of the unit.
        node_selector (ServerCreateResourceUnitRequestNodeSelector | Unset): Node labels workloads using this unit must
            match.
    """

    limits: ServerCreateResourceUnitRequestLimits
    name: str
    requests: ServerCreateResourceUnitRequestRequests
    annotations: ServerCreateResourceUnitRequestAnnotations | Unset = UNSET
    description: str | Unset = UNSET
    node_selector: ServerCreateResourceUnitRequestNodeSelector | Unset = UNSET
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
        from ..models.server_create_resource_unit_request_annotations import (
            ServerCreateResourceUnitRequestAnnotations,
        )
        from ..models.server_create_resource_unit_request_limits import (
            ServerCreateResourceUnitRequestLimits,
        )
        from ..models.server_create_resource_unit_request_node_selector import (
            ServerCreateResourceUnitRequestNodeSelector,
        )
        from ..models.server_create_resource_unit_request_requests import (
            ServerCreateResourceUnitRequestRequests,
        )

        d = dict(src_dict)
        limits = ServerCreateResourceUnitRequestLimits.from_dict(d.pop("limits"))

        name = d.pop("name")

        requests = ServerCreateResourceUnitRequestRequests.from_dict(d.pop("requests"))

        _annotations = d.pop("annotations", UNSET)
        annotations: ServerCreateResourceUnitRequestAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = ServerCreateResourceUnitRequestAnnotations.from_dict(
                _annotations
            )

        description = d.pop("description", UNSET)

        _node_selector = d.pop("nodeSelector", UNSET)
        node_selector: ServerCreateResourceUnitRequestNodeSelector | Unset
        if isinstance(_node_selector, Unset):
            node_selector = UNSET
        else:
            node_selector = ServerCreateResourceUnitRequestNodeSelector.from_dict(
                _node_selector
            )

        server_create_resource_unit_request = cls(
            limits=limits,
            name=name,
            requests=requests,
            annotations=annotations,
            description=description,
            node_selector=node_selector,
        )

        server_create_resource_unit_request.additional_properties = d
        return server_create_resource_unit_request

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
