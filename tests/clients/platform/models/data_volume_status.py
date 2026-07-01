from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.data_volume_mount import DataVolumeMount


T = TypeVar("T", bound="DataVolumeStatus")


@_attrs_define
class DataVolumeStatus:
    """
    Attributes:
        bound_capacity (str | Unset): Actually bound capacity once the volume is Bound.
        mounts (list[DataVolumeMount] | Unset): Workloads currently mounting this volume (populated on detail get).
        phase (str | Unset): PVC phase: Pending, Bound, or Lost.
        used_bytes (int | Unset): Best-effort used bytes from the monitoring stack; omitted when unavailable.
    """

    bound_capacity: str | Unset = UNSET
    mounts: list[DataVolumeMount] | Unset = UNSET
    phase: str | Unset = UNSET
    used_bytes: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        bound_capacity = self.bound_capacity

        mounts: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.mounts, Unset):
            mounts = []
            for mounts_item_data in self.mounts:
                mounts_item = mounts_item_data.to_dict()
                mounts.append(mounts_item)

        phase = self.phase

        used_bytes = self.used_bytes

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if bound_capacity is not UNSET:
            field_dict["boundCapacity"] = bound_capacity
        if mounts is not UNSET:
            field_dict["mounts"] = mounts
        if phase is not UNSET:
            field_dict["phase"] = phase
        if used_bytes is not UNSET:
            field_dict["usedBytes"] = used_bytes

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.data_volume_mount import DataVolumeMount

        d = dict(src_dict)
        bound_capacity = d.pop("boundCapacity", UNSET)

        _mounts = d.pop("mounts", UNSET)
        mounts: list[DataVolumeMount] | Unset = UNSET
        if _mounts is not UNSET:
            mounts = []
            for mounts_item_data in _mounts:
                mounts_item = DataVolumeMount.from_dict(mounts_item_data)

                mounts.append(mounts_item)

        phase = d.pop("phase", UNSET)

        used_bytes = d.pop("usedBytes", UNSET)

        data_volume_status = cls(
            bound_capacity=bound_capacity,
            mounts=mounts,
            phase=phase,
            used_bytes=used_bytes,
        )

        data_volume_status.additional_properties = d
        return data_volume_status

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
