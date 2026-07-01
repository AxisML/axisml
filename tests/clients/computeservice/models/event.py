from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.metav_1_time import Metav1Time


T = TypeVar("T", bound="Event")


@_attrs_define
class Event:
    """
    Example:
        {'eventTime': '2026-06-28T09:30:00Z', 'note': 'Successfully assigned team-vision/resnet-train-7-worker-0 to gpu-
            node-a100-03', 'object': 'Pod/resnet-train-7-worker-0', 'reason': 'Scheduled', 'reportingController': 'axisml-
            scheduler', 'type': 'Normal'}

    Attributes:
        object_ (str): Target object as "<kind>/<name>".
        reason (str): Short machine-readable reason for the event (e.g. Scheduled, Pulled).
        type_ (str): Event type (Normal or Warning).
        event_time (Metav1Time | None | Unset): Time the event was first observed.
        note (str | Unset): Human-readable description of the event.
        reporting_controller (str | Unset): Controller that reported the event.
    """

    object_: str
    reason: str
    type_: str
    event_time: Metav1Time | None | Unset = UNSET
    note: str | Unset = UNSET
    reporting_controller: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.metav_1_time import Metav1Time

        object_ = self.object_

        reason = self.reason

        type_ = self.type_

        event_time: dict[str, Any] | None | Unset
        if isinstance(self.event_time, Unset):
            event_time = UNSET
        elif isinstance(self.event_time, Metav1Time):
            event_time = self.event_time.to_dict()
        else:
            event_time = self.event_time

        note = self.note

        reporting_controller = self.reporting_controller

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "object": object_,
                "reason": reason,
                "type": type_,
            }
        )
        if event_time is not UNSET:
            field_dict["eventTime"] = event_time
        if note is not UNSET:
            field_dict["note"] = note
        if reporting_controller is not UNSET:
            field_dict["reportingController"] = reporting_controller

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.metav_1_time import Metav1Time

        d = dict(src_dict)
        object_ = d.pop("object")

        reason = d.pop("reason")

        type_ = d.pop("type")

        def _parse_event_time(data: object) -> Metav1Time | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                event_time_type_1 = Metav1Time.from_dict(data)

                return event_time_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Metav1Time | None | Unset, data)

        event_time = _parse_event_time(d.pop("eventTime", UNSET))

        note = d.pop("note", UNSET)

        reporting_controller = d.pop("reportingController", UNSET)

        event = cls(
            object_=object_,
            reason=reason,
            type_=type_,
            event_time=event_time,
            note=note,
            reporting_controller=reporting_controller,
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
