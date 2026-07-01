from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.server_quota_status import ServerQuotaStatus


T = TypeVar("T", bound="ServerTenantStatus")


@_attrs_define
class ServerTenantStatus:
    """
    Attributes:
        message (str | Unset): Human-readable detail about the current phase.
        namespace_ready (bool | Unset): Whether the tenant's namespace has been provisioned.
        observed_generation (int | Unset): Generation of the spec the operator last reconciled.
        phase (str | Unset): Current reconciliation phase reported by the operator.
        quotas (list[ServerQuotaStatus] | Unset): Per-pool quota readiness and live usage.
    """

    message: str | Unset = UNSET
    namespace_ready: bool | Unset = UNSET
    observed_generation: int | Unset = UNSET
    phase: str | Unset = UNSET
    quotas: list[ServerQuotaStatus] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        message = self.message

        namespace_ready = self.namespace_ready

        observed_generation = self.observed_generation

        phase = self.phase

        quotas: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.quotas, Unset):
            quotas = []
            for quotas_item_data in self.quotas:
                quotas_item = quotas_item_data.to_dict()
                quotas.append(quotas_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if message is not UNSET:
            field_dict["message"] = message
        if namespace_ready is not UNSET:
            field_dict["namespaceReady"] = namespace_ready
        if observed_generation is not UNSET:
            field_dict["observedGeneration"] = observed_generation
        if phase is not UNSET:
            field_dict["phase"] = phase
        if quotas is not UNSET:
            field_dict["quotas"] = quotas

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.server_quota_status import ServerQuotaStatus

        d = dict(src_dict)
        message = d.pop("message", UNSET)

        namespace_ready = d.pop("namespaceReady", UNSET)

        observed_generation = d.pop("observedGeneration", UNSET)

        phase = d.pop("phase", UNSET)

        _quotas = d.pop("quotas", UNSET)
        quotas: list[ServerQuotaStatus] | Unset = UNSET
        if _quotas is not UNSET:
            quotas = []
            for quotas_item_data in _quotas:
                quotas_item = ServerQuotaStatus.from_dict(quotas_item_data)

                quotas.append(quotas_item)

        server_tenant_status = cls(
            message=message,
            namespace_ready=namespace_ready,
            observed_generation=observed_generation,
            phase=phase,
            quotas=quotas,
        )

        server_tenant_status.additional_properties = d
        return server_tenant_status

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
