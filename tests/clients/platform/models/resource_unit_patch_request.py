from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.resource_map import ResourceMap
    from ..models.string_map import StringMap
    from ..models.toleration import Toleration


T = TypeVar("T", bound="ResourceUnitPatchRequest")


@_attrs_define
class ResourceUnitPatchRequest:
    """
    Example:
        {'description': 'Updated 2x A100 GPU compute unit.', 'limits': {'cpu': '24', 'memory': '192Gi',
            'nvidia.com/gpu': '2'}}

    Attributes:
        annotations (StringMap | Unset):
        description (str | Unset): Updated free-text unit description.
        limits (ResourceMap | Unset): Kubernetes-style resource quantity map (e.g., {"cpu": "100", "memory": "1Ti",
            "nvidia.com/gpu": "8"}).
        node_selector (StringMap | Unset):
        requests (ResourceMap | Unset): Kubernetes-style resource quantity map (e.g., {"cpu": "100", "memory": "1Ti",
            "nvidia.com/gpu": "8"}).
        tolerations (list[Toleration] | Unset): Replacement toleration set.
    """

    annotations: StringMap | Unset = UNSET
    description: str | Unset = UNSET
    limits: ResourceMap | Unset = UNSET
    node_selector: StringMap | Unset = UNSET
    requests: ResourceMap | Unset = UNSET
    tolerations: list[Toleration] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

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

        tolerations: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.tolerations, Unset):
            tolerations = []
            for tolerations_item_data in self.tolerations:
                tolerations_item = tolerations_item_data.to_dict()
                tolerations.append(tolerations_item)

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
        if tolerations is not UNSET:
            field_dict["tolerations"] = tolerations

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.resource_map import ResourceMap
        from ..models.string_map import StringMap
        from ..models.toleration import Toleration

        d = dict(src_dict)
        _annotations = d.pop("annotations", UNSET)
        annotations: StringMap | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = StringMap.from_dict(_annotations)

        description = d.pop("description", UNSET)

        _limits = d.pop("limits", UNSET)
        limits: ResourceMap | Unset
        if isinstance(_limits, Unset):
            limits = UNSET
        else:
            limits = ResourceMap.from_dict(_limits)

        _node_selector = d.pop("nodeSelector", UNSET)
        node_selector: StringMap | Unset
        if isinstance(_node_selector, Unset):
            node_selector = UNSET
        else:
            node_selector = StringMap.from_dict(_node_selector)

        _requests = d.pop("requests", UNSET)
        requests: ResourceMap | Unset
        if isinstance(_requests, Unset):
            requests = UNSET
        else:
            requests = ResourceMap.from_dict(_requests)

        _tolerations = d.pop("tolerations", UNSET)
        tolerations: list[Toleration] | Unset = UNSET
        if _tolerations is not UNSET:
            tolerations = []
            for tolerations_item_data in _tolerations:
                tolerations_item = Toleration.from_dict(tolerations_item_data)

                tolerations.append(tolerations_item)

        resource_unit_patch_request = cls(
            annotations=annotations,
            description=description,
            limits=limits,
            node_selector=node_selector,
            requests=requests,
            tolerations=tolerations,
        )

        resource_unit_patch_request.additional_properties = d
        return resource_unit_patch_request

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
