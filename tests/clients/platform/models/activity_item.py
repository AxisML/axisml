from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="ActivityItem")


@_attrs_define
class ActivityItem:
    """
    Example:
        {'action': 'succeeded', 'actor': 'li.wei', 'id': 'act-9f3a2e5c', 'kind': 'run', 'name': 'resnet-train-7',
            'phase': 'Succeeded', 'timestamp': '2026-06-28T09:25:00Z'}

    Attributes:
        action (str): What happened (created, started, stopped, succeeded, failed, deleted, ...).
        id (str): Stable activity entry identifier.
        kind (str): Subject resource kind (workspace, job, experiment, run, mlservice, trafficpolicy).
        name (str): Subject resource name.
        timestamp (datetime.datetime): Time the activity occurred.
        actor (str | Unset): Username that triggered the activity.
        phase (str | Unset): Subject's phase at the time of the entry.
    """

    action: str
    id: str
    kind: str
    name: str
    timestamp: datetime.datetime
    actor: str | Unset = UNSET
    phase: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        action = self.action

        id = self.id

        kind = self.kind

        name = self.name

        timestamp = self.timestamp.isoformat()

        actor = self.actor

        phase = self.phase

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "action": action,
                "id": id,
                "kind": kind,
                "name": name,
                "timestamp": timestamp,
            }
        )
        if actor is not UNSET:
            field_dict["actor"] = actor
        if phase is not UNSET:
            field_dict["phase"] = phase

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        action = d.pop("action")

        id = d.pop("id")

        kind = d.pop("kind")

        name = d.pop("name")

        timestamp = datetime.datetime.fromisoformat(d.pop("timestamp"))

        actor = d.pop("actor", UNSET)

        phase = d.pop("phase", UNSET)

        activity_item = cls(
            action=action,
            id=id,
            kind=kind,
            name=name,
            timestamp=timestamp,
            actor=actor,
            phase=phase,
        )

        activity_item.additional_properties = d
        return activity_item

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
