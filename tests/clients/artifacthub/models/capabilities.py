from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="Capabilities")


@_attrs_define
class Capabilities:
    """
    Example:
        {'kinds': ['model', 'dataset', 'image'], 'upload': True}

    Attributes:
        kinds (list[str]): Artifact kinds served in this deployment form (model, dataset, image).
        upload (bool): Whether two-phase artifact upload is available in this deployment form.
    """

    kinds: list[str]
    upload: bool
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        kinds = self.kinds

        upload = self.upload

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "kinds": kinds,
                "upload": upload,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kinds = cast(list[str], d.pop("kinds"))

        upload = d.pop("upload")

        capabilities = cls(
            kinds=kinds,
            upload=upload,
        )

        capabilities.additional_properties = d
        return capabilities

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
