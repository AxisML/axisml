from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_config_map_env_source import Corev1ConfigMapEnvSource
    from ..models.corev_1_secret_env_source import Corev1SecretEnvSource


T = TypeVar("T", bound="Corev1EnvFromSource")


@_attrs_define
class Corev1EnvFromSource:
    """
    Attributes:
        config_map_ref (Corev1ConfigMapEnvSource | None | Unset):
        prefix (str | Unset):
        secret_ref (Corev1SecretEnvSource | None | Unset):
    """

    config_map_ref: Corev1ConfigMapEnvSource | None | Unset = UNSET
    prefix: str | Unset = UNSET
    secret_ref: Corev1SecretEnvSource | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.corev_1_config_map_env_source import Corev1ConfigMapEnvSource
        from ..models.corev_1_secret_env_source import Corev1SecretEnvSource

        config_map_ref: dict[str, Any] | None | Unset
        if isinstance(self.config_map_ref, Unset):
            config_map_ref = UNSET
        elif isinstance(self.config_map_ref, Corev1ConfigMapEnvSource):
            config_map_ref = self.config_map_ref.to_dict()
        else:
            config_map_ref = self.config_map_ref

        prefix = self.prefix

        secret_ref: dict[str, Any] | None | Unset
        if isinstance(self.secret_ref, Unset):
            secret_ref = UNSET
        elif isinstance(self.secret_ref, Corev1SecretEnvSource):
            secret_ref = self.secret_ref.to_dict()
        else:
            secret_ref = self.secret_ref

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if config_map_ref is not UNSET:
            field_dict["configMapRef"] = config_map_ref
        if prefix is not UNSET:
            field_dict["prefix"] = prefix
        if secret_ref is not UNSET:
            field_dict["secretRef"] = secret_ref

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_config_map_env_source import Corev1ConfigMapEnvSource
        from ..models.corev_1_secret_env_source import Corev1SecretEnvSource

        d = dict(src_dict)

        def _parse_config_map_ref(
            data: object,
        ) -> Corev1ConfigMapEnvSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                config_map_ref_type_1 = Corev1ConfigMapEnvSource.from_dict(data)

                return config_map_ref_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1ConfigMapEnvSource | None | Unset, data)

        config_map_ref = _parse_config_map_ref(d.pop("configMapRef", UNSET))

        prefix = d.pop("prefix", UNSET)

        def _parse_secret_ref(data: object) -> Corev1SecretEnvSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                secret_ref_type_1 = Corev1SecretEnvSource.from_dict(data)

                return secret_ref_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1SecretEnvSource | None | Unset, data)

        secret_ref = _parse_secret_ref(d.pop("secretRef", UNSET))

        corev_1_env_from_source = cls(
            config_map_ref=config_map_ref,
            prefix=prefix,
            secret_ref=secret_ref,
        )

        corev_1_env_from_source.additional_properties = d
        return corev_1_env_from_source

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
