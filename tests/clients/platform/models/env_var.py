from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.env_var_value_from import EnvVarValueFrom


T = TypeVar("T", bound="EnvVar")


@_attrs_define
class EnvVar:
    """
    Example:
        {'name': 'NCCL_DEBUG', 'value': 'INFO'}

    Attributes:
        name (str): Environment variable name.
        value (str | Unset): Literal value for the variable.
        value_from (EnvVarValueFrom | Unset): Source the value from a secret/configmap/field (pass-through to the K8s
            EnvVarSource shape).
    """

    name: str
    value: str | Unset = UNSET
    value_from: EnvVarValueFrom | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        value = self.value

        value_from: dict[str, Any] | Unset = UNSET
        if not isinstance(self.value_from, Unset):
            value_from = self.value_from.to_dict()

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
        from ..models.env_var_value_from import EnvVarValueFrom

        d = dict(src_dict)
        name = d.pop("name")

        value = d.pop("value", UNSET)

        _value_from = d.pop("valueFrom", UNSET)
        value_from: EnvVarValueFrom | Unset
        if isinstance(_value_from, Unset):
            value_from = UNSET
        else:
            value_from = EnvVarValueFrom.from_dict(_value_from)

        env_var = cls(
            name=name,
            value=value,
            value_from=value_from,
        )

        env_var.additional_properties = d
        return env_var

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
