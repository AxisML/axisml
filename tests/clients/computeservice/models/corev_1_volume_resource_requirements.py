from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_volume_resource_requirements_limits import (
        Corev1VolumeResourceRequirementsLimits,
    )
    from ..models.corev_1_volume_resource_requirements_requests import (
        Corev1VolumeResourceRequirementsRequests,
    )


T = TypeVar("T", bound="Corev1VolumeResourceRequirements")


@_attrs_define
class Corev1VolumeResourceRequirements:
    """
    Attributes:
        limits (Corev1VolumeResourceRequirementsLimits | Unset): Map of resource name (cpu, memory, nvidia.com/gpu, …)
            to resource.Quantity.
        requests (Corev1VolumeResourceRequirementsRequests | Unset): Map of resource name (cpu, memory, nvidia.com/gpu,
            …) to resource.Quantity.
    """

    limits: Corev1VolumeResourceRequirementsLimits | Unset = UNSET
    requests: Corev1VolumeResourceRequirementsRequests | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        limits: dict[str, Any] | Unset = UNSET
        if not isinstance(self.limits, Unset):
            limits = self.limits.to_dict()

        requests: dict[str, Any] | Unset = UNSET
        if not isinstance(self.requests, Unset):
            requests = self.requests.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if limits is not UNSET:
            field_dict["limits"] = limits
        if requests is not UNSET:
            field_dict["requests"] = requests

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_volume_resource_requirements_limits import (
            Corev1VolumeResourceRequirementsLimits,
        )
        from ..models.corev_1_volume_resource_requirements_requests import (
            Corev1VolumeResourceRequirementsRequests,
        )

        d = dict(src_dict)
        _limits = d.pop("limits", UNSET)
        limits: Corev1VolumeResourceRequirementsLimits | Unset
        if isinstance(_limits, Unset):
            limits = UNSET
        else:
            limits = Corev1VolumeResourceRequirementsLimits.from_dict(_limits)

        _requests = d.pop("requests", UNSET)
        requests: Corev1VolumeResourceRequirementsRequests | Unset
        if isinstance(_requests, Unset):
            requests = UNSET
        else:
            requests = Corev1VolumeResourceRequirementsRequests.from_dict(_requests)

        corev_1_volume_resource_requirements = cls(
            limits=limits,
            requests=requests,
        )

        corev_1_volume_resource_requirements.additional_properties = d
        return corev_1_volume_resource_requirements

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
