from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="MLRunRunPolicySpec")


@_attrs_define
class MLRunRunPolicySpec:
    """
    Attributes:
        active_deadline_seconds (int | None | Unset):
        backoff_limit (int | None | Unset):
        suspend (bool | Unset):
        ttl_seconds_after_finished (int | None | Unset):
    """

    active_deadline_seconds: int | None | Unset = UNSET
    backoff_limit: int | None | Unset = UNSET
    suspend: bool | Unset = UNSET
    ttl_seconds_after_finished: int | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        active_deadline_seconds: int | None | Unset
        if isinstance(self.active_deadline_seconds, Unset):
            active_deadline_seconds = UNSET
        else:
            active_deadline_seconds = self.active_deadline_seconds

        backoff_limit: int | None | Unset
        if isinstance(self.backoff_limit, Unset):
            backoff_limit = UNSET
        else:
            backoff_limit = self.backoff_limit

        suspend = self.suspend

        ttl_seconds_after_finished: int | None | Unset
        if isinstance(self.ttl_seconds_after_finished, Unset):
            ttl_seconds_after_finished = UNSET
        else:
            ttl_seconds_after_finished = self.ttl_seconds_after_finished

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if active_deadline_seconds is not UNSET:
            field_dict["activeDeadlineSeconds"] = active_deadline_seconds
        if backoff_limit is not UNSET:
            field_dict["backoffLimit"] = backoff_limit
        if suspend is not UNSET:
            field_dict["suspend"] = suspend
        if ttl_seconds_after_finished is not UNSET:
            field_dict["ttlSecondsAfterFinished"] = ttl_seconds_after_finished

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)

        def _parse_active_deadline_seconds(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        active_deadline_seconds = _parse_active_deadline_seconds(
            d.pop("activeDeadlineSeconds", UNSET)
        )

        def _parse_backoff_limit(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        backoff_limit = _parse_backoff_limit(d.pop("backoffLimit", UNSET))

        suspend = d.pop("suspend", UNSET)

        def _parse_ttl_seconds_after_finished(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        ttl_seconds_after_finished = _parse_ttl_seconds_after_finished(
            d.pop("ttlSecondsAfterFinished", UNSET)
        )

        ml_run_run_policy_spec = cls(
            active_deadline_seconds=active_deadline_seconds,
            backoff_limit=backoff_limit,
            suspend=suspend,
            ttl_seconds_after_finished=ttl_seconds_after_finished,
        )

        ml_run_run_policy_spec.additional_properties = d
        return ml_run_run_policy_spec

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
