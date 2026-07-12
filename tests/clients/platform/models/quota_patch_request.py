from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.quota_resources import QuotaResources
    from ..models.quota_unit import QuotaUnit


T = TypeVar("T", bound="QuotaPatchRequest")


@_attrs_define
class QuotaPatchRequest:
    """
    Example:
        {'units': [{'quantity': 6, 'unitName': 'a100-2x'}]}

    Attributes:
        quota (None | QuotaResources | Unset): Replacement direct min/max resources for the pool. Mutually exclusive
            with units.
        units (list[QuotaUnit] | Unset): Replacement per-unit allocations for the pool. Mutually exclusive with quota.
    """

    quota: None | QuotaResources | Unset = UNSET
    units: list[QuotaUnit] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.quota_resources import QuotaResources

        quota: dict[str, Any] | None | Unset
        if isinstance(self.quota, Unset):
            quota = UNSET
        elif isinstance(self.quota, QuotaResources):
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
        field_dict.update({})
        if quota is not UNSET:
            field_dict["quota"] = quota
        if units is not UNSET:
            field_dict["units"] = units

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.quota_resources import QuotaResources
        from ..models.quota_unit import QuotaUnit

        d = dict(src_dict)

        def _parse_quota(data: object) -> None | QuotaResources | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                quota_type_1 = QuotaResources.from_dict(data)

                return quota_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | QuotaResources | Unset, data)

        quota = _parse_quota(d.pop("quota", UNSET))

        _units = d.pop("units", UNSET)
        units: list[QuotaUnit] | Unset = UNSET
        if _units is not UNSET:
            units = []
            for units_item_data in _units:
                units_item = QuotaUnit.from_dict(units_item_data)

                units.append(units_item)

        quota_patch_request = cls(
            quota=quota,
            units=units,
        )

        quota_patch_request.additional_properties = d
        return quota_patch_request

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
