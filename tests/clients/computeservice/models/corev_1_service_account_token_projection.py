from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Corev1ServiceAccountTokenProjection")


@_attrs_define
class Corev1ServiceAccountTokenProjection:
    """
    Attributes:
        path (str):
        audience (str | Unset):
        expiration_seconds (int | None | Unset):
    """

    path: str
    audience: str | Unset = UNSET
    expiration_seconds: int | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        path = self.path

        audience = self.audience

        expiration_seconds: int | None | Unset
        if isinstance(self.expiration_seconds, Unset):
            expiration_seconds = UNSET
        else:
            expiration_seconds = self.expiration_seconds

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "path": path,
            }
        )
        if audience is not UNSET:
            field_dict["audience"] = audience
        if expiration_seconds is not UNSET:
            field_dict["expirationSeconds"] = expiration_seconds

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        path = d.pop("path")

        audience = d.pop("audience", UNSET)

        def _parse_expiration_seconds(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        expiration_seconds = _parse_expiration_seconds(
            d.pop("expirationSeconds", UNSET)
        )

        corev_1_service_account_token_projection = cls(
            path=path,
            audience=audience,
            expiration_seconds=expiration_seconds,
        )

        corev_1_service_account_token_projection.additional_properties = d
        return corev_1_service_account_token_projection

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
