from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="MLServiceRunPolicy")


@_attrs_define
class MLServiceRunPolicy:
    """
    Attributes:
        progress_deadline_seconds (int | None | Unset):
    """

    progress_deadline_seconds: int | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        progress_deadline_seconds: int | None | Unset
        if isinstance(self.progress_deadline_seconds, Unset):
            progress_deadline_seconds = UNSET
        else:
            progress_deadline_seconds = self.progress_deadline_seconds

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if progress_deadline_seconds is not UNSET:
            field_dict["progressDeadlineSeconds"] = progress_deadline_seconds

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)

        def _parse_progress_deadline_seconds(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        progress_deadline_seconds = _parse_progress_deadline_seconds(
            d.pop("progressDeadlineSeconds", UNSET)
        )

        ml_service_run_policy = cls(
            progress_deadline_seconds=progress_deadline_seconds,
        )

        ml_service_run_policy.additional_properties = d
        return ml_service_run_policy

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
