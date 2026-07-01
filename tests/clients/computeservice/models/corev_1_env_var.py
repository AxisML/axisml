from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_env_var_source import Corev1EnvVarSource


T = TypeVar("T", bound="Corev1EnvVar")


@_attrs_define
class Corev1EnvVar:
    """
    Attributes:
        name (str):
        value (str | Unset):
        value_from (Corev1EnvVarSource | None | Unset):
    """

    name: str
    value: str | Unset = UNSET
    value_from: Corev1EnvVarSource | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.corev_1_env_var_source import Corev1EnvVarSource

        name = self.name

        value = self.value

        value_from: dict[str, Any] | None | Unset
        if isinstance(self.value_from, Unset):
            value_from = UNSET
        elif isinstance(self.value_from, Corev1EnvVarSource):
            value_from = self.value_from.to_dict()
        else:
            value_from = self.value_from

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
            }
        )
        if value is not UNSET:
            field_dict["value"] = value
        if value_from is not UNSET:
            field_dict["valueFrom"] = value_from

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_env_var_source import Corev1EnvVarSource

        d = dict(src_dict)
        name = d.pop("name")

        value = d.pop("value", UNSET)

        def _parse_value_from(data: object) -> Corev1EnvVarSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                value_from_type_1 = Corev1EnvVarSource.from_dict(data)

                return value_from_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1EnvVarSource | None | Unset, data)

        value_from = _parse_value_from(d.pop("valueFrom", UNSET))

        corev_1_env_var = cls(
            name=name,
            value=value,
            value_from=value_from,
        )

        corev_1_env_var.additional_properties = d
        return corev_1_env_var

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
