from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.event_type import EventType
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.event_involved_object import EventInvolvedObject


T = TypeVar("T", bound="Event")


@_attrs_define
class Event:
    """
    Example:
        {'count': 1, 'firstTimestamp': '2026-06-28T09:00:00Z', 'involvedObject': {'kind': 'Pod', 'name': 'resnet-
            train-7-worker-0', 'namespace': 'axisml-team-vision'}, 'lastTimestamp': '2026-06-28T09:00:00Z', 'message':
            'Successfully assigned resnet-train-7-worker-0 to gpu-node-03.', 'reason': 'Scheduled', 'source': 'default-
            scheduler', 'type': 'Normal'}

    Attributes:
        last_timestamp (datetime.datetime): Time the event was last observed.
        message (str): Human-readable event message.
        reason (str): Short machine-readable reason for the event.
        type_ (EventType): Event type (Normal, Warning).
        count (int | Unset): Number of times this event has occurred.
        first_timestamp (datetime.datetime | Unset): Time the event was first observed.
        involved_object (EventInvolvedObject | Unset): Object the event is about.
        source (str | Unset): Component that reported the event.
    """

    last_timestamp: datetime.datetime
    message: str
    reason: str
    type_: EventType
    count: int | Unset = UNSET
    first_timestamp: datetime.datetime | Unset = UNSET
    involved_object: EventInvolvedObject | Unset = UNSET
    source: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        last_timestamp = self.last_timestamp.isoformat()

        message = self.message

        reason = self.reason

        type_ = self.type_.value

        count = self.count

        first_timestamp: str | Unset = UNSET
        if not isinstance(self.first_timestamp, Unset):
            first_timestamp = self.first_timestamp.isoformat()

        involved_object: dict[str, Any] | Unset = UNSET
        if not isinstance(self.involved_object, Unset):
            involved_object = self.involved_object.to_dict()

        source = self.source

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "lastTimestamp": last_timestamp,
                "message": message,
                "reason": reason,
                "type": type_,
            }
        )
        if count is not UNSET:
            field_dict["count"] = count
        if first_timestamp is not UNSET:
            field_dict["firstTimestamp"] = first_timestamp
        if involved_object is not UNSET:
            field_dict["involvedObject"] = involved_object
        if source is not UNSET:
            field_dict["source"] = source

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.event_involved_object import EventInvolvedObject

        d = dict(src_dict)
        last_timestamp = datetime.datetime.fromisoformat(d.pop("lastTimestamp"))

        message = d.pop("message")

        reason = d.pop("reason")

        type_ = EventType(d.pop("type"))

        count = d.pop("count", UNSET)

        _first_timestamp = d.pop("firstTimestamp", UNSET)
        first_timestamp: datetime.datetime | Unset
        if isinstance(_first_timestamp, Unset):
            first_timestamp = UNSET
        else:
            first_timestamp = datetime.datetime.fromisoformat(_first_timestamp)

        _involved_object = d.pop("involvedObject", UNSET)
        involved_object: EventInvolvedObject | Unset
        if isinstance(_involved_object, Unset):
            involved_object = UNSET
        else:
            involved_object = EventInvolvedObject.from_dict(_involved_object)

        source = d.pop("source", UNSET)

        event = cls(
            last_timestamp=last_timestamp,
            message=message,
            reason=reason,
            type_=type_,
            count=count,
            first_timestamp=first_timestamp,
            involved_object=involved_object,
            source=source,
        )

        event.additional_properties = d
        return event

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
