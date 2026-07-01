from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_resource_claim import Corev1ResourceClaim
    from ..models.corev_1_resource_requirements_limits import (
        Corev1ResourceRequirementsLimits,
    )
    from ..models.corev_1_resource_requirements_requests import (
        Corev1ResourceRequirementsRequests,
    )


T = TypeVar("T", bound="Corev1ResourceRequirements")


@_attrs_define
class Corev1ResourceRequirements:
    """
    Attributes:
        claims (list[Corev1ResourceClaim] | Unset):
        limits (Corev1ResourceRequirementsLimits | Unset): Map of resource name (cpu, memory, nvidia.com/gpu, …) to
            resource.Quantity.
        requests (Corev1ResourceRequirementsRequests | Unset): Map of resource name (cpu, memory, nvidia.com/gpu, …) to
            resource.Quantity.
    """

    claims: list[Corev1ResourceClaim] | Unset = UNSET
    limits: Corev1ResourceRequirementsLimits | Unset = UNSET
    requests: Corev1ResourceRequirementsRequests | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        claims: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.claims, Unset):
            claims = []
            for claims_item_data in self.claims:
                claims_item = claims_item_data.to_dict()
                claims.append(claims_item)

        limits: dict[str, Any] | Unset = UNSET
        if not isinstance(self.limits, Unset):
            limits = self.limits.to_dict()

        requests: dict[str, Any] | Unset = UNSET
        if not isinstance(self.requests, Unset):
            requests = self.requests.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if claims is not UNSET:
            field_dict["claims"] = claims
        if limits is not UNSET:
            field_dict["limits"] = limits
        if requests is not UNSET:
            field_dict["requests"] = requests

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_resource_claim import Corev1ResourceClaim
        from ..models.corev_1_resource_requirements_limits import (
            Corev1ResourceRequirementsLimits,
        )
        from ..models.corev_1_resource_requirements_requests import (
            Corev1ResourceRequirementsRequests,
        )

        d = dict(src_dict)
        _claims = d.pop("claims", UNSET)
        claims: list[Corev1ResourceClaim] | Unset = UNSET
        if _claims is not UNSET:
            claims = []
            for claims_item_data in _claims:
                claims_item = Corev1ResourceClaim.from_dict(claims_item_data)

                claims.append(claims_item)

        _limits = d.pop("limits", UNSET)
        limits: Corev1ResourceRequirementsLimits | Unset
        if isinstance(_limits, Unset):
            limits = UNSET
        else:
            limits = Corev1ResourceRequirementsLimits.from_dict(_limits)

        _requests = d.pop("requests", UNSET)
        requests: Corev1ResourceRequirementsRequests | Unset
        if isinstance(_requests, Unset):
            requests = UNSET
        else:
            requests = Corev1ResourceRequirementsRequests.from_dict(_requests)

        corev_1_resource_requirements = cls(
            claims=claims,
            limits=limits,
            requests=requests,
        )

        corev_1_resource_requirements.additional_properties = d
        return corev_1_resource_requirements

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
