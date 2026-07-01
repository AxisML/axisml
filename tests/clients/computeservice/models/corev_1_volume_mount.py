from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Corev1VolumeMount")


@_attrs_define
class Corev1VolumeMount:
    """
    Attributes:
        mount_path (str):
        name (str):
        mount_propagation (None | str | Unset):
        read_only (bool | Unset):
        recursive_read_only (None | str | Unset):
        sub_path (str | Unset):
        sub_path_expr (str | Unset):
    """

    mount_path: str
    name: str
    mount_propagation: None | str | Unset = UNSET
    read_only: bool | Unset = UNSET
    recursive_read_only: None | str | Unset = UNSET
    sub_path: str | Unset = UNSET
    sub_path_expr: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        mount_path = self.mount_path

        name = self.name

        mount_propagation: None | str | Unset
        if isinstance(self.mount_propagation, Unset):
            mount_propagation = UNSET
        else:
            mount_propagation = self.mount_propagation

        read_only = self.read_only

        recursive_read_only: None | str | Unset
        if isinstance(self.recursive_read_only, Unset):
            recursive_read_only = UNSET
        else:
            recursive_read_only = self.recursive_read_only

        sub_path = self.sub_path

        sub_path_expr = self.sub_path_expr

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "mountPath": mount_path,
                "name": name,
            }
        )
        if mount_propagation is not UNSET:
            field_dict["mountPropagation"] = mount_propagation
        if read_only is not UNSET:
            field_dict["readOnly"] = read_only
        if recursive_read_only is not UNSET:
            field_dict["recursiveReadOnly"] = recursive_read_only
        if sub_path is not UNSET:
            field_dict["subPath"] = sub_path
        if sub_path_expr is not UNSET:
            field_dict["subPathExpr"] = sub_path_expr

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mount_path = d.pop("mountPath")

        name = d.pop("name")

        def _parse_mount_propagation(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        mount_propagation = _parse_mount_propagation(d.pop("mountPropagation", UNSET))

        read_only = d.pop("readOnly", UNSET)

        def _parse_recursive_read_only(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        recursive_read_only = _parse_recursive_read_only(
            d.pop("recursiveReadOnly", UNSET)
        )

        sub_path = d.pop("subPath", UNSET)

        sub_path_expr = d.pop("subPathExpr", UNSET)

        corev_1_volume_mount = cls(
            mount_path=mount_path,
            name=name,
            mount_propagation=mount_propagation,
            read_only=read_only,
            recursive_read_only=recursive_read_only,
            sub_path=sub_path,
            sub_path_expr=sub_path_expr,
        )

        corev_1_volume_mount.additional_properties = d
        return corev_1_volume_mount

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
