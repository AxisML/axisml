from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.metav_1_fields_v1 import Metav1FieldsV1
    from ..models.metav_1_time import Metav1Time


T = TypeVar("T", bound="Metav1ManagedFieldsEntry")


@_attrs_define
class Metav1ManagedFieldsEntry:
    """
    Attributes:
        api_version (str | Unset):
        fields_type (str | Unset):
        fields_v1 (Metav1FieldsV1 | None | Unset):
        manager (str | Unset):
        operation (str | Unset):
        subresource (str | Unset):
        time (Metav1Time | None | Unset):
    """

    api_version: str | Unset = UNSET
    fields_type: str | Unset = UNSET
    fields_v1: Metav1FieldsV1 | None | Unset = UNSET
    manager: str | Unset = UNSET
    operation: str | Unset = UNSET
    subresource: str | Unset = UNSET
    time: Metav1Time | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.metav_1_fields_v1 import Metav1FieldsV1
        from ..models.metav_1_time import Metav1Time

        api_version = self.api_version

        fields_type = self.fields_type

        fields_v1: dict[str, Any] | None | Unset
        if isinstance(self.fields_v1, Unset):
            fields_v1 = UNSET
        elif isinstance(self.fields_v1, Metav1FieldsV1):
            fields_v1 = self.fields_v1.to_dict()
        else:
            fields_v1 = self.fields_v1

        manager = self.manager

        operation = self.operation

        subresource = self.subresource

        time: dict[str, Any] | None | Unset
        if isinstance(self.time, Unset):
            time = UNSET
        elif isinstance(self.time, Metav1Time):
            time = self.time.to_dict()
        else:
            time = self.time

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if api_version is not UNSET:
            field_dict["apiVersion"] = api_version
        if fields_type is not UNSET:
            field_dict["fieldsType"] = fields_type
        if fields_v1 is not UNSET:
            field_dict["fieldsV1"] = fields_v1
        if manager is not UNSET:
            field_dict["manager"] = manager
        if operation is not UNSET:
            field_dict["operation"] = operation
        if subresource is not UNSET:
            field_dict["subresource"] = subresource
        if time is not UNSET:
            field_dict["time"] = time

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.metav_1_fields_v1 import Metav1FieldsV1
        from ..models.metav_1_time import Metav1Time

        d = dict(src_dict)
        api_version = d.pop("apiVersion", UNSET)

        fields_type = d.pop("fieldsType", UNSET)

        def _parse_fields_v1(data: object) -> Metav1FieldsV1 | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                fields_v1_type_1 = Metav1FieldsV1.from_dict(data)

                return fields_v1_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Metav1FieldsV1 | None | Unset, data)

        fields_v1 = _parse_fields_v1(d.pop("fieldsV1", UNSET))

        manager = d.pop("manager", UNSET)

        operation = d.pop("operation", UNSET)

        subresource = d.pop("subresource", UNSET)

        def _parse_time(data: object) -> Metav1Time | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                time_type_1 = Metav1Time.from_dict(data)

                return time_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Metav1Time | None | Unset, data)

        time = _parse_time(d.pop("time", UNSET))

        metav_1_managed_fields_entry = cls(
            api_version=api_version,
            fields_type=fields_type,
            fields_v1=fields_v1,
            manager=manager,
            operation=operation,
            subresource=subresource,
            time=time,
        )

        metav_1_managed_fields_entry.additional_properties = d
        return metav_1_managed_fields_entry

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
