from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="RunPolicy")


@_attrs_define
class RunPolicy:
    """
    Example:
        {'activeDeadlineSeconds': 86400, 'backoffLimit': 2, 'progressDeadlineSeconds': 600, 'ttlSecondsAfterFinished':
            3600}

    Attributes:
        active_deadline_seconds (int | Unset): Hard wall-clock limit (seconds) before the run is terminated.
        backoff_limit (int | Unset): Number of retries before the run is marked failed.
        progress_deadline_seconds (int | Unset): Seconds without progress before the run is considered stalled.
        ttl_seconds_after_finished (int | Unset): Seconds to retain a finished run before garbage collection.
    """

    active_deadline_seconds: int | Unset = UNSET
    backoff_limit: int | Unset = UNSET
    progress_deadline_seconds: int | Unset = UNSET
    ttl_seconds_after_finished: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        active_deadline_seconds = self.active_deadline_seconds

        backoff_limit = self.backoff_limit

        progress_deadline_seconds = self.progress_deadline_seconds

        ttl_seconds_after_finished = self.ttl_seconds_after_finished

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if active_deadline_seconds is not UNSET:
            field_dict["activeDeadlineSeconds"] = active_deadline_seconds
        if backoff_limit is not UNSET:
            field_dict["backoffLimit"] = backoff_limit
        if progress_deadline_seconds is not UNSET:
            field_dict["progressDeadlineSeconds"] = progress_deadline_seconds
        if ttl_seconds_after_finished is not UNSET:
            field_dict["ttlSecondsAfterFinished"] = ttl_seconds_after_finished

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        active_deadline_seconds = d.pop("activeDeadlineSeconds", UNSET)

        backoff_limit = d.pop("backoffLimit", UNSET)

        progress_deadline_seconds = d.pop("progressDeadlineSeconds", UNSET)

        ttl_seconds_after_finished = d.pop("ttlSecondsAfterFinished", UNSET)

        run_policy = cls(
            active_deadline_seconds=active_deadline_seconds,
            backoff_limit=backoff_limit,
            progress_deadline_seconds=progress_deadline_seconds,
            ttl_seconds_after_finished=ttl_seconds_after_finished,
        )

        run_policy.additional_properties = d
        return run_policy

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
