from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.server_quota_resources import ServerQuotaResources
    from ..models.server_quota_unit import ServerQuotaUnit


T = TypeVar("T", bound="ServerQuota")


@_attrs_define
class ServerQuota:
    """
    Attributes:
        pool (str): ResourcePool this quota applies to.
        quota (None | ServerQuotaResources | Unset): Direct min/max resources for this pool. Mutually exclusive with
            units.
        units (list[ServerQuotaUnit] | Unset): Unit × quantity selections granted to the tenant under this pool.
            Mutually exclusive with quota.
    """

    pool: str
    quota: None | ServerQuotaResources | Unset = UNSET
    units: list[ServerQuotaUnit] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.server_quota_resources import ServerQuotaResources

        pool = self.pool

        quota: dict[str, Any] | None | Unset
        if isinstance(self.quota, Unset):
            quota = UNSET
        elif isinstance(self.quota, ServerQuotaResources):
            quota = self.quota.to_dict()
        else:
            quota = self.quota

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
                "pool": pool,
            }
        )
        if quota is not UNSET:
            field_dict["quota"] = quota
        if units is not UNSET:
            field_dict["units"] = units

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.server_quota_resources import ServerQuotaResources
        from ..models.server_quota_unit import ServerQuotaUnit

        d = dict(src_dict)
        pool = d.pop("pool")

        def _parse_quota(data: object) -> None | ServerQuotaResources | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                quota_type_1 = ServerQuotaResources.from_dict(data)

                return quota_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | ServerQuotaResources | Unset, data)

        quota = _parse_quota(d.pop("quota", UNSET))

        _units = d.pop("units", UNSET)
        units: list[ServerQuotaUnit] | Unset = UNSET
        if _units is not UNSET:
            units = []
            for units_item_data in _units:
                units_item = ServerQuotaUnit.from_dict(units_item_data)

                units.append(units_item)

        server_quota = cls(
            pool=pool,
            quota=quota,
            units=units,
        )

        server_quota.additional_properties = d
        return server_quota

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
