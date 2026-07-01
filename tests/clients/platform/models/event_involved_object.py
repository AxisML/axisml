from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="EventInvolvedObject")


@_attrs_define
class EventInvolvedObject:
    """Object the event is about.

    Attributes:
        kind (str | Unset): Kind of the object the event refers to.
        name (str | Unset): Name of the object the event refers to.
        namespace (str | Unset): Namespace of the object the event refers to.
    """

    kind: str | Unset = UNSET
    name: str | Unset = UNSET
    namespace: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        kind = self.kind

        name = self.name

        namespace = self.namespace

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if kind is not UNSET:
            field_dict["kind"] = kind
        if name is not UNSET:
            field_dict["name"] = name
        if namespace is not UNSET:
            field_dict["namespace"] = namespace

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = d.pop("kind", UNSET)

        name = d.pop("name", UNSET)

        namespace = d.pop("namespace", UNSET)

        event_involved_object = cls(
            kind=kind,
            name=name,
            namespace=namespace,
        )

        event_involved_object.additional_properties = d
        return event_involved_object

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
