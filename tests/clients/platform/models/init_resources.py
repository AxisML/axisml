from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.config_map_init import ConfigMapInit
    from ..models.image_pull_secret_init import ImagePullSecretInit
    from ..models.secret_init import SecretInit
    from ..models.service_account_init import ServiceAccountInit


T = TypeVar("T", bound="InitResources")


@_attrs_define
class InitResources:
    """
    Example:
        {'configMaps': [{'name': 'shared-config', 'sourceConfigMapRef': {'name': 'shared-config', 'namespace': 'axisml-
            system'}}], 'imagePullSecrets': [{'name': 'registry-pull', 'sourceSecretRef': {'name': 'registry-pull',
            'namespace': 'axisml-system'}}], 'secrets': [{'name': 'wandb-api-key', 'sourceSecretRef': {'name': 'wandb-api-
            key', 'namespace': 'axisml-system'}, 'type': 'Opaque'}], 'serviceAccounts': [{'imagePullSecrets': ['registry-
            pull'], 'name': 'trainer', 'rbac': {'roleRefs': [{'kind': 'ClusterRole', 'name': 'edit'}]}}]}

    Attributes:
        config_maps (list[ConfigMapInit] | Unset): ConfigMaps to seed in the tenant namespace.
        image_pull_secrets (list[ImagePullSecretInit] | Unset): Image pull Secrets to seed in the tenant namespace.
        secrets (list[SecretInit] | Unset): Secrets to seed in the tenant namespace.
        service_accounts (list[ServiceAccountInit] | Unset): ServiceAccounts to seed in the tenant namespace.
    """

    config_maps: list[ConfigMapInit] | Unset = UNSET
    image_pull_secrets: list[ImagePullSecretInit] | Unset = UNSET
    secrets: list[SecretInit] | Unset = UNSET
    service_accounts: list[ServiceAccountInit] | Unset = UNSET
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

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.config_map_init import ConfigMapInit
        from ..models.image_pull_secret_init import ImagePullSecretInit
        from ..models.secret_init import SecretInit
        from ..models.service_account_init import ServiceAccountInit

        d = dict(src_dict)
        _config_maps = d.pop("configMaps", UNSET)
        config_maps: list[ConfigMapInit] | Unset = UNSET
        if _config_maps is not UNSET:
            config_maps = []
            for config_maps_item_data in _config_maps:
                config_maps_item = ConfigMapInit.from_dict(config_maps_item_data)

                config_maps.append(config_maps_item)

        _image_pull_secrets = d.pop("imagePullSecrets", UNSET)
        image_pull_secrets: list[ImagePullSecretInit] | Unset = UNSET
        if _image_pull_secrets is not UNSET:
            image_pull_secrets = []
            for image_pull_secrets_item_data in _image_pull_secrets:
                image_pull_secrets_item = ImagePullSecretInit.from_dict(
                    image_pull_secrets_item_data
                )

                image_pull_secrets.append(image_pull_secrets_item)

        _secrets = d.pop("secrets", UNSET)
        secrets: list[SecretInit] | Unset = UNSET
        if _secrets is not UNSET:
            secrets = []
            for secrets_item_data in _secrets:
                secrets_item = SecretInit.from_dict(secrets_item_data)

                secrets.append(secrets_item)

        _service_accounts = d.pop("serviceAccounts", UNSET)
        service_accounts: list[ServiceAccountInit] | Unset = UNSET
        if _service_accounts is not UNSET:
            service_accounts = []
            for service_accounts_item_data in _service_accounts:
                service_accounts_item = ServiceAccountInit.from_dict(
                    service_accounts_item_data
                )

                service_accounts.append(service_accounts_item)

        init_resources = cls(
            config_maps=config_maps,
            image_pull_secrets=image_pull_secrets,
            secrets=secrets,
            service_accounts=service_accounts,
        )

        init_resources.additional_properties = d
        return init_resources

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
