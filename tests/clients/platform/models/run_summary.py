from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.run_phase import RunPhase
from ..types import UNSET, Unset

T = TypeVar("T", bound="RunSummary")


@_attrs_define
class RunSummary:
    """
    Example:
        {'active': 1, 'count': 7, 'latestPhase': 'Running', 'latestRunAt': '2026-06-28T09:00:00Z', 'recent':
            ['Succeeded', 'Failed', 'Succeeded', 'Succeeded', 'Running']}

    Attributes:
        count (int): Total number of Runs of the definition.
        active (int | Unset): Runs currently in a non-terminal phase.
        latest_phase (RunPhase | Unset): Run (compute MLRun) phase. The active (non-terminal) phases — Creating /
            Pending / Running / Canceling — block Job-definition deletion.
        latest_run_at (datetime.datetime | None | Unset): Creation time of the most recent Run.
        recent (list[RunPhase] | Unset): Most-recent Run phases, oldest-to-newest, for the status strip.
    """

    count: int
    active: int | Unset = UNSET
    latest_phase: RunPhase | Unset = UNSET
    latest_run_at: datetime.datetime | None | Unset = UNSET
    recent: list[RunPhase] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        count = self.count

        active = self.active

        latest_phase: str | Unset = UNSET
        if not isinstance(self.latest_phase, Unset):
            latest_phase = self.latest_phase.value

        latest_run_at: None | str | Unset
        if isinstance(self.latest_run_at, Unset):
            latest_run_at = UNSET
        elif isinstance(self.latest_run_at, datetime.datetime):
            latest_run_at = self.latest_run_at.isoformat()
        else:
            latest_run_at = self.latest_run_at

        recent: list[str] | Unset = UNSET
        if not isinstance(self.recent, Unset):
            recent = []
            for recent_item_data in self.recent:
                recent_item = recent_item_data.value
                recent.append(recent_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "count": count,
            }
        )
        if active is not UNSET:
            field_dict["active"] = active
        if latest_phase is not UNSET:
            field_dict["latestPhase"] = latest_phase
        if latest_run_at is not UNSET:
            field_dict["latestRunAt"] = latest_run_at
        if recent is not UNSET:
            field_dict["recent"] = recent

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        active = d.pop("active", UNSET)

        _latest_phase = d.pop("latestPhase", UNSET)
        latest_phase: RunPhase | Unset
        if isinstance(_latest_phase, Unset):
            latest_phase = UNSET
        else:
            latest_phase = RunPhase(_latest_phase)

        def _parse_latest_run_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                latest_run_at_type_0 = datetime.datetime.fromisoformat(data)

                return latest_run_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        latest_run_at = _parse_latest_run_at(d.pop("latestRunAt", UNSET))

        _recent = d.pop("recent", UNSET)
        recent: list[RunPhase] | Unset = UNSET
        if _recent is not UNSET:
            recent = []
            for recent_item_data in _recent:
                recent_item = RunPhase(recent_item_data)

                recent.append(recent_item)

        run_summary = cls(
            count=count,
            active=active,
            latest_phase=latest_phase,
            latest_run_at=latest_run_at,
            recent=recent,
        )

        run_summary.additional_properties = d
        return run_summary

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
