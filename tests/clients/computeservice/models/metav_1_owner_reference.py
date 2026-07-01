from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Metav1OwnerReference")


@_attrs_define
class Metav1OwnerReference:
    """
    Attributes:
        api_version (str):
        kind (str):
        name (str):
        uid (str):
        block_owner_deletion (bool | None | Unset):
        controller (bool | None | Unset):
    """

    api_version: str
    kind: str
    name: str
    uid: str
    block_owner_deletion: bool | None | Unset = UNSET
    controller: bool | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        api_version = self.api_version

        kind = self.kind

        name = self.name

        uid = self.uid

        block_owner_deletion: bool | None | Unset
        if isinstance(self.block_owner_deletion, Unset):
            block_owner_deletion = UNSET
        else:
            block_owner_deletion = self.block_owner_deletion

        controller: bool | None | Unset
        if isinstance(self.controller, Unset):
            controller = UNSET
        else:
            controller = self.controller

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "apiVersion": api_version,
                "kind": kind,
                "name": name,
                "uid": uid,
            }
        )
        if block_owner_deletion is not UNSET:
            field_dict["blockOwnerDeletion"] = block_owner_deletion
        if controller is not UNSET:
            field_dict["controller"] = controller

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        api_version = d.pop("apiVersion")

        kind = d.pop("kind")

        name = d.pop("name")

        uid = d.pop("uid")

        def _parse_block_owner_deletion(data: object) -> bool | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(bool | None | Unset, data)

        block_owner_deletion = _parse_block_owner_deletion(
            d.pop("blockOwnerDeletion", UNSET)
        )

        def _parse_controller(data: object) -> bool | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(bool | None | Unset, data)

        controller = _parse_controller(d.pop("controller", UNSET))

        metav_1_owner_reference = cls(
            api_version=api_version,
            kind=kind,
            name=name,
            uid=uid,
            block_owner_deletion=block_owner_deletion,
            controller=controller,
        )

        metav_1_owner_reference.additional_properties = d
        return metav_1_owner_reference

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
