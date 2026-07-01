from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.patch_resource_unit_request_annotations import (
        PatchResourceUnitRequestAnnotations,
    )
    from ..models.patch_resource_unit_request_limits import (
        PatchResourceUnitRequestLimits,
    )
    from ..models.patch_resource_unit_request_node_selector import (
        PatchResourceUnitRequestNodeSelector,
    )
    from ..models.patch_resource_unit_request_requests import (
        PatchResourceUnitRequestRequests,
    )


T = TypeVar("T", bound="PatchResourceUnitRequest")


@_attrs_define
class PatchResourceUnitRequest:
    """
    Example:
        {'description': '2× A100 GPU compute unit (updated).', 'limits': {'cpu': '24', 'memory': '192Gi',
            'nvidia.com/gpu': '2'}}

    Attributes:
        annotations (PatchResourceUnitRequestAnnotations | Unset): Replacement annotations for the unit.
        description (None | str | Unset): New description; omit to leave unchanged, empty string to clear.
        limits (PatchResourceUnitRequestLimits | Unset): Replacement resource limits for the unit.
        node_selector (PatchResourceUnitRequestNodeSelector | Unset): Replacement node selector for the unit.
        requests (PatchResourceUnitRequestRequests | Unset): Replacement resource requests for the unit.
    """

    annotations: PatchResourceUnitRequestAnnotations | Unset = UNSET
    description: None | str | Unset = UNSET
    limits: PatchResourceUnitRequestLimits | Unset = UNSET
    node_selector: PatchResourceUnitRequestNodeSelector | Unset = UNSET
    requests: PatchResourceUnitRequestRequests | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        description: None | str | Unset
        if isinstance(self.description, Unset):
            description = UNSET
        else:
            description = self.description

        limits: dict[str, Any] | Unset = UNSET
        if not isinstance(self.limits, Unset):
            limits = self.limits.to_dict()

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
        if node_selector is not UNSET:
            field_dict["nodeSelector"] = node_selector
        if requests is not UNSET:
            field_dict["requests"] = requests

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.patch_resource_unit_request_annotations import (
            PatchResourceUnitRequestAnnotations,
        )
        from ..models.patch_resource_unit_request_limits import (
            PatchResourceUnitRequestLimits,
        )
        from ..models.patch_resource_unit_request_node_selector import (
            PatchResourceUnitRequestNodeSelector,
        )
        from ..models.patch_resource_unit_request_requests import (
            PatchResourceUnitRequestRequests,
        )

        d = dict(src_dict)
        _annotations = d.pop("annotations", UNSET)
        annotations: PatchResourceUnitRequestAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = PatchResourceUnitRequestAnnotations.from_dict(_annotations)

        def _parse_description(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        description = _parse_description(d.pop("description", UNSET))

        _limits = d.pop("limits", UNSET)
        limits: PatchResourceUnitRequestLimits | Unset
        if isinstance(_limits, Unset):
            limits = UNSET
        else:
            limits = PatchResourceUnitRequestLimits.from_dict(_limits)

        _node_selector = d.pop("nodeSelector", UNSET)
        node_selector: PatchResourceUnitRequestNodeSelector | Unset
        if isinstance(_node_selector, Unset):
            node_selector = UNSET
        else:
            node_selector = PatchResourceUnitRequestNodeSelector.from_dict(
                _node_selector
            )

        _requests = d.pop("requests", UNSET)
        requests: PatchResourceUnitRequestRequests | Unset
        if isinstance(_requests, Unset):
            requests = UNSET
        else:
            requests = PatchResourceUnitRequestRequests.from_dict(_requests)

        patch_resource_unit_request = cls(
            annotations=annotations,
            description=description,
            limits=limits,
            node_selector=node_selector,
            requests=requests,
        )

        patch_resource_unit_request.additional_properties = d
        return patch_resource_unit_request

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
