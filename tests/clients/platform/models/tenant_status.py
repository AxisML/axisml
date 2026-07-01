from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.condition import Condition
    from ..models.quota_status import QuotaStatus


T = TypeVar("T", bound="TenantStatus")


@_attrs_define
class TenantStatus:
    """
    Example:
        {'conditions': [{'lastTransitionTime': '2026-06-28T09:30:00Z', 'message': 'Namespace and quota provisioned.',
            'reason': 'Provisioned', 'status': 'True', 'type': 'Ready'}], 'message': 'Tenant is active.', 'quotas':
            [{'pool': 'gpu-a100', 'units': [{'quantity': 4, 'unitName': 'a100-2x', 'used': 3}]}]}

    Attributes:
        conditions (list[Condition] | Unset): Live status conditions reported by cluster-manager.
        message (str | Unset): Human-readable status detail for the tenant.
        quotas (list[QuotaStatus] | Unset): Live per-pool quota usage.
    """

    conditions: list[Condition] | Unset = UNSET
    message: str | Unset = UNSET
    quotas: list[QuotaStatus] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        conditions: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.conditions, Unset):
            conditions = []
            for conditions_item_data in self.conditions:
                conditions_item = conditions_item_data.to_dict()
                conditions.append(conditions_item)

        message = self.message

        quotas: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.quotas, Unset):
            quotas = []
            for quotas_item_data in self.quotas:
                quotas_item = quotas_item_data.to_dict()
                quotas.append(quotas_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if conditions is not UNSET:
            field_dict["conditions"] = conditions
        if message is not UNSET:
            field_dict["message"] = message
        if quotas is not UNSET:
            field_dict["quotas"] = quotas

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.condition import Condition
        from ..models.quota_status import QuotaStatus

        d = dict(src_dict)
        _conditions = d.pop("conditions", UNSET)
        conditions: list[Condition] | Unset = UNSET
        if _conditions is not UNSET:
            conditions = []
            for conditions_item_data in _conditions:
                conditions_item = Condition.from_dict(conditions_item_data)

                conditions.append(conditions_item)

        message = d.pop("message", UNSET)

        _quotas = d.pop("quotas", UNSET)
        quotas: list[QuotaStatus] | Unset = UNSET
        if _quotas is not UNSET:
            quotas = []
            for quotas_item_data in _quotas:
                quotas_item = QuotaStatus.from_dict(quotas_item_data)

                quotas.append(quotas_item)

        tenant_status = cls(
            conditions=conditions,
            message=message,
            quotas=quotas,
        )

        tenant_status.additional_properties = d
        return tenant_status

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
