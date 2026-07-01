from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_config_map_key_selector import Corev1ConfigMapKeySelector
    from ..models.corev_1_file_key_selector import Corev1FileKeySelector
    from ..models.corev_1_object_field_selector import Corev1ObjectFieldSelector
    from ..models.corev_1_resource_field_selector import Corev1ResourceFieldSelector
    from ..models.corev_1_secret_key_selector import Corev1SecretKeySelector


T = TypeVar("T", bound="Corev1EnvVarSource")


@_attrs_define
class Corev1EnvVarSource:
    """
    Attributes:
        config_map_key_ref (Corev1ConfigMapKeySelector | None | Unset):
        field_ref (Corev1ObjectFieldSelector | None | Unset):
        file_key_ref (Corev1FileKeySelector | None | Unset):
        resource_field_ref (Corev1ResourceFieldSelector | None | Unset):
        secret_key_ref (Corev1SecretKeySelector | None | Unset):
    """

    config_map_key_ref: Corev1ConfigMapKeySelector | None | Unset = UNSET
    field_ref: Corev1ObjectFieldSelector | None | Unset = UNSET
    file_key_ref: Corev1FileKeySelector | None | Unset = UNSET
    resource_field_ref: Corev1ResourceFieldSelector | None | Unset = UNSET
    secret_key_ref: Corev1SecretKeySelector | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.corev_1_config_map_key_selector import Corev1ConfigMapKeySelector
        from ..models.corev_1_file_key_selector import Corev1FileKeySelector
        from ..models.corev_1_object_field_selector import Corev1ObjectFieldSelector
        from ..models.corev_1_resource_field_selector import Corev1ResourceFieldSelector
        from ..models.corev_1_secret_key_selector import Corev1SecretKeySelector

        config_map_key_ref: dict[str, Any] | None | Unset
        if isinstance(self.config_map_key_ref, Unset):
            config_map_key_ref = UNSET
        elif isinstance(self.config_map_key_ref, Corev1ConfigMapKeySelector):
            config_map_key_ref = self.config_map_key_ref.to_dict()
        else:
            config_map_key_ref = self.config_map_key_ref

        field_ref: dict[str, Any] | None | Unset
        if isinstance(self.field_ref, Unset):
            field_ref = UNSET
        elif isinstance(self.field_ref, Corev1ObjectFieldSelector):
            field_ref = self.field_ref.to_dict()
        else:
            field_ref = self.field_ref

        file_key_ref: dict[str, Any] | None | Unset
        if isinstance(self.file_key_ref, Unset):
            file_key_ref = UNSET
        elif isinstance(self.file_key_ref, Corev1FileKeySelector):
            file_key_ref = self.file_key_ref.to_dict()
        else:
            file_key_ref = self.file_key_ref

        resource_field_ref: dict[str, Any] | None | Unset
        if isinstance(self.resource_field_ref, Unset):
            resource_field_ref = UNSET
        elif isinstance(self.resource_field_ref, Corev1ResourceFieldSelector):
            resource_field_ref = self.resource_field_ref.to_dict()
        else:
            resource_field_ref = self.resource_field_ref

        secret_key_ref: dict[str, Any] | None | Unset
        if isinstance(self.secret_key_ref, Unset):
            secret_key_ref = UNSET
        elif isinstance(self.secret_key_ref, Corev1SecretKeySelector):
            secret_key_ref = self.secret_key_ref.to_dict()
        else:
            secret_key_ref = self.secret_key_ref

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if config_map_key_ref is not UNSET:
            field_dict["configMapKeyRef"] = config_map_key_ref
        if field_ref is not UNSET:
            field_dict["fieldRef"] = field_ref
        if file_key_ref is not UNSET:
            field_dict["fileKeyRef"] = file_key_ref
        if resource_field_ref is not UNSET:
            field_dict["resourceFieldRef"] = resource_field_ref
        if secret_key_ref is not UNSET:
            field_dict["secretKeyRef"] = secret_key_ref

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_config_map_key_selector import Corev1ConfigMapKeySelector
        from ..models.corev_1_file_key_selector import Corev1FileKeySelector
        from ..models.corev_1_object_field_selector import Corev1ObjectFieldSelector
        from ..models.corev_1_resource_field_selector import Corev1ResourceFieldSelector
        from ..models.corev_1_secret_key_selector import Corev1SecretKeySelector

        d = dict(src_dict)

        def _parse_config_map_key_ref(
            data: object,
        ) -> Corev1ConfigMapKeySelector | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                config_map_key_ref_type_1 = Corev1ConfigMapKeySelector.from_dict(data)

                return config_map_key_ref_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1ConfigMapKeySelector | None | Unset, data)

        config_map_key_ref = _parse_config_map_key_ref(d.pop("configMapKeyRef", UNSET))

        def _parse_field_ref(data: object) -> Corev1ObjectFieldSelector | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                field_ref_type_1 = Corev1ObjectFieldSelector.from_dict(data)

                return field_ref_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1ObjectFieldSelector | None | Unset, data)

        field_ref = _parse_field_ref(d.pop("fieldRef", UNSET))

        def _parse_file_key_ref(data: object) -> Corev1FileKeySelector | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                file_key_ref_type_1 = Corev1FileKeySelector.from_dict(data)

                return file_key_ref_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1FileKeySelector | None | Unset, data)

        file_key_ref = _parse_file_key_ref(d.pop("fileKeyRef", UNSET))

        def _parse_resource_field_ref(
            data: object,
        ) -> Corev1ResourceFieldSelector | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                resource_field_ref_type_1 = Corev1ResourceFieldSelector.from_dict(data)

                return resource_field_ref_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1ResourceFieldSelector | None | Unset, data)

        resource_field_ref = _parse_resource_field_ref(d.pop("resourceFieldRef", UNSET))

        def _parse_secret_key_ref(
            data: object,
        ) -> Corev1SecretKeySelector | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                secret_key_ref_type_1 = Corev1SecretKeySelector.from_dict(data)

                return secret_key_ref_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1SecretKeySelector | None | Unset, data)

        secret_key_ref = _parse_secret_key_ref(d.pop("secretKeyRef", UNSET))

        corev_1_env_var_source = cls(
            config_map_key_ref=config_map_key_ref,
            field_ref=field_ref,
            file_key_ref=file_key_ref,
            resource_field_ref=resource_field_ref,
            secret_key_ref=secret_key_ref,
        )

        corev_1_env_var_source.additional_properties = d
        return corev_1_env_var_source

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
