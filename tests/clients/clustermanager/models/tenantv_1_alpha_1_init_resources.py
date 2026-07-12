from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.tenantv_1_alpha_1_config_map_spec import Tenantv1Alpha1ConfigMapSpec
    from ..models.tenantv_1_alpha_1_image_pull_secret_spec import (
        Tenantv1Alpha1ImagePullSecretSpec,
    )
    from ..models.tenantv_1_alpha_1_secret_spec import Tenantv1Alpha1SecretSpec
    from ..models.tenantv_1_alpha_1_service_account_spec import (
        Tenantv1Alpha1ServiceAccountSpec,
    )
    from ..models.tenantv_1_alpha_1_volume_spec import Tenantv1Alpha1VolumeSpec


T = TypeVar("T", bound="Tenantv1Alpha1InitResources")


@_attrs_define
class Tenantv1Alpha1InitResources:
    """
    Attributes:
        config_maps (list[Tenantv1Alpha1ConfigMapSpec] | Unset):
        image_pull_secrets (list[Tenantv1Alpha1ImagePullSecretSpec] | Unset):
        secrets (list[Tenantv1Alpha1SecretSpec] | Unset):
        service_accounts (list[Tenantv1Alpha1ServiceAccountSpec] | Unset):
        volumes (list[Tenantv1Alpha1VolumeSpec] | Unset):
    """

    config_maps: list[Tenantv1Alpha1ConfigMapSpec] | Unset = UNSET
    image_pull_secrets: list[Tenantv1Alpha1ImagePullSecretSpec] | Unset = UNSET
    secrets: list[Tenantv1Alpha1SecretSpec] | Unset = UNSET
    service_accounts: list[Tenantv1Alpha1ServiceAccountSpec] | Unset = UNSET
    volumes: list[Tenantv1Alpha1VolumeSpec] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        config_maps: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.config_maps, Unset):
            config_maps = []
            for config_maps_item_data in self.config_maps:
                config_maps_item = config_maps_item_data.to_dict()
                config_maps.append(config_maps_item)

        image_pull_secrets: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.image_pull_secrets, Unset):
            image_pull_secrets = []
            for image_pull_secrets_item_data in self.image_pull_secrets:
                image_pull_secrets_item = image_pull_secrets_item_data.to_dict()
                image_pull_secrets.append(image_pull_secrets_item)

        secrets: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.secrets, Unset):
            secrets = []
            for secrets_item_data in self.secrets:
                secrets_item = secrets_item_data.to_dict()
                secrets.append(secrets_item)

        service_accounts: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.service_accounts, Unset):
            service_accounts = []
            for service_accounts_item_data in self.service_accounts:
                service_accounts_item = service_accounts_item_data.to_dict()
                service_accounts.append(service_accounts_item)

        volumes: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.volumes, Unset):
            volumes = []
            for volumes_item_data in self.volumes:
                volumes_item = volumes_item_data.to_dict()
                volumes.append(volumes_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if config_maps is not UNSET:
            field_dict["configMaps"] = config_maps
        if image_pull_secrets is not UNSET:
            field_dict["imagePullSecrets"] = image_pull_secrets
        if secrets is not UNSET:
            field_dict["secrets"] = secrets
        if service_accounts is not UNSET:
            field_dict["serviceAccounts"] = service_accounts
        if volumes is not UNSET:
            field_dict["volumes"] = volumes

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.tenantv_1_alpha_1_config_map_spec import (
            Tenantv1Alpha1ConfigMapSpec,
        )
        from ..models.tenantv_1_alpha_1_image_pull_secret_spec import (
            Tenantv1Alpha1ImagePullSecretSpec,
        )
        from ..models.tenantv_1_alpha_1_secret_spec import Tenantv1Alpha1SecretSpec
        from ..models.tenantv_1_alpha_1_service_account_spec import (
            Tenantv1Alpha1ServiceAccountSpec,
        )
        from ..models.tenantv_1_alpha_1_volume_spec import Tenantv1Alpha1VolumeSpec

        d = dict(src_dict)
        _config_maps = d.pop("configMaps", UNSET)
        config_maps: list[Tenantv1Alpha1ConfigMapSpec] | Unset = UNSET
        if _config_maps is not UNSET:
            config_maps = []
            for config_maps_item_data in _config_maps:
                config_maps_item = Tenantv1Alpha1ConfigMapSpec.from_dict(
                    config_maps_item_data
                )

                config_maps.append(config_maps_item)

        _image_pull_secrets = d.pop("imagePullSecrets", UNSET)
        image_pull_secrets: list[Tenantv1Alpha1ImagePullSecretSpec] | Unset = UNSET
        if _image_pull_secrets is not UNSET:
            image_pull_secrets = []
            for image_pull_secrets_item_data in _image_pull_secrets:
                image_pull_secrets_item = Tenantv1Alpha1ImagePullSecretSpec.from_dict(
                    image_pull_secrets_item_data
                )

                image_pull_secrets.append(image_pull_secrets_item)

        _secrets = d.pop("secrets", UNSET)
        secrets: list[Tenantv1Alpha1SecretSpec] | Unset = UNSET
        if _secrets is not UNSET:
            secrets = []
            for secrets_item_data in _secrets:
                secrets_item = Tenantv1Alpha1SecretSpec.from_dict(secrets_item_data)

                secrets.append(secrets_item)

        _service_accounts = d.pop("serviceAccounts", UNSET)
        service_accounts: list[Tenantv1Alpha1ServiceAccountSpec] | Unset = UNSET
        if _service_accounts is not UNSET:
            service_accounts = []
            for service_accounts_item_data in _service_accounts:
                service_accounts_item = Tenantv1Alpha1ServiceAccountSpec.from_dict(
                    service_accounts_item_data
                )

                service_accounts.append(service_accounts_item)

        _volumes = d.pop("volumes", UNSET)
        volumes: list[Tenantv1Alpha1VolumeSpec] | Unset = UNSET
        if _volumes is not UNSET:
            volumes = []
            for volumes_item_data in _volumes:
                volumes_item = Tenantv1Alpha1VolumeSpec.from_dict(volumes_item_data)

                volumes.append(volumes_item)

        tenantv_1_alpha_1_init_resources = cls(
            config_maps=config_maps,
            image_pull_secrets=image_pull_secrets,
            secrets=secrets,
            service_accounts=service_accounts,
            volumes=volumes,
        )

        tenantv_1_alpha_1_init_resources.additional_properties = d
        return tenantv_1_alpha_1_init_resources

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
