from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.condition_status import ConditionStatus
from ..types import UNSET, Unset

T = TypeVar("T", bound="Condition")


@_attrs_define
class Condition:
    """
    Example:
        {'lastTransitionTime': '2026-06-28T09:30:00Z', 'message': 'All worker replicas are ready.', 'reason':
            'AllReplicasReady', 'status': 'True', 'type': 'Ready'}

    Attributes:
        status (ConditionStatus): Status of the condition (True, False, Unknown).
        type_ (str): Condition type (e.g. Ready, Available).
        last_transition_time (datetime.datetime | Unset): Time the condition last transitioned to this status.
        message (str | Unset): Human-readable detail about the last transition.
        reason (str | Unset): Machine-readable reason for the condition's last transition.
    """

    status: ConditionStatus
    type_: str
    last_transition_time: datetime.datetime | Unset = UNSET
    message: str | Unset = UNSET
    reason: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        status = self.status.value

        type_ = self.type_

        last_transition_time: str | Unset = UNSET
        if not isinstance(self.last_transition_time, Unset):
            last_transition_time = self.last_transition_time.isoformat()

        message = self.message

        reason = self.reason

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "status": status,
                "type": type_,
            }
        )
        if last_transition_time is not UNSET:
            field_dict["lastTransitionTime"] = last_transition_time
        if message is not UNSET:
            field_dict["message"] = message
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        status = ConditionStatus(d.pop("status"))

        type_ = d.pop("type")

        _last_transition_time = d.pop("lastTransitionTime", UNSET)
        last_transition_time: datetime.datetime | Unset
        if isinstance(_last_transition_time, Unset):
            last_transition_time = UNSET
        else:
            last_transition_time = datetime.datetime.fromisoformat(
                _last_transition_time
            )

        message = d.pop("message", UNSET)

        reason = d.pop("reason", UNSET)

        condition = cls(
            status=status,
            type_=type_,
            last_transition_time=last_transition_time,
            message=message,
            reason=reason,
        )

        condition.additional_properties = d
        return condition

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
