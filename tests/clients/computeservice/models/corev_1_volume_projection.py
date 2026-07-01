from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_cluster_trust_bundle_projection import (
        Corev1ClusterTrustBundleProjection,
    )
    from ..models.corev_1_config_map_projection import Corev1ConfigMapProjection
    from ..models.corev_1_downward_api_projection import Corev1DownwardAPIProjection
    from ..models.corev_1_pod_certificate_projection import (
        Corev1PodCertificateProjection,
    )
    from ..models.corev_1_secret_projection import Corev1SecretProjection
    from ..models.corev_1_service_account_token_projection import (
        Corev1ServiceAccountTokenProjection,
    )


T = TypeVar("T", bound="Corev1VolumeProjection")


@_attrs_define
class Corev1VolumeProjection:
    """
    Attributes:
        cluster_trust_bundle (Corev1ClusterTrustBundleProjection | None | Unset):
        config_map (Corev1ConfigMapProjection | None | Unset):
        downward_api (Corev1DownwardAPIProjection | None | Unset):
        pod_certificate (Corev1PodCertificateProjection | None | Unset):
        secret (Corev1SecretProjection | None | Unset):
        service_account_token (Corev1ServiceAccountTokenProjection | None | Unset):
    """

    cluster_trust_bundle: Corev1ClusterTrustBundleProjection | None | Unset = UNSET
    config_map: Corev1ConfigMapProjection | None | Unset = UNSET
    downward_api: Corev1DownwardAPIProjection | None | Unset = UNSET
    pod_certificate: Corev1PodCertificateProjection | None | Unset = UNSET
    secret: Corev1SecretProjection | None | Unset = UNSET
    service_account_token: Corev1ServiceAccountTokenProjection | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.corev_1_cluster_trust_bundle_projection import (
            Corev1ClusterTrustBundleProjection,
        )
        from ..models.corev_1_config_map_projection import Corev1ConfigMapProjection
        from ..models.corev_1_downward_api_projection import Corev1DownwardAPIProjection
        from ..models.corev_1_pod_certificate_projection import (
            Corev1PodCertificateProjection,
        )
        from ..models.corev_1_secret_projection import Corev1SecretProjection
        from ..models.corev_1_service_account_token_projection import (
            Corev1ServiceAccountTokenProjection,
        )

        cluster_trust_bundle: dict[str, Any] | None | Unset
        if isinstance(self.cluster_trust_bundle, Unset):
            cluster_trust_bundle = UNSET
        elif isinstance(self.cluster_trust_bundle, Corev1ClusterTrustBundleProjection):
            cluster_trust_bundle = self.cluster_trust_bundle.to_dict()
        else:
            cluster_trust_bundle = self.cluster_trust_bundle

        config_map: dict[str, Any] | None | Unset
        if isinstance(self.config_map, Unset):
            config_map = UNSET
        elif isinstance(self.config_map, Corev1ConfigMapProjection):
            config_map = self.config_map.to_dict()
        else:
            config_map = self.config_map

        downward_api: dict[str, Any] | None | Unset
        if isinstance(self.downward_api, Unset):
            downward_api = UNSET
        elif isinstance(self.downward_api, Corev1DownwardAPIProjection):
            downward_api = self.downward_api.to_dict()
        else:
            downward_api = self.downward_api

        pod_certificate: dict[str, Any] | None | Unset
        if isinstance(self.pod_certificate, Unset):
            pod_certificate = UNSET
        elif isinstance(self.pod_certificate, Corev1PodCertificateProjection):
            pod_certificate = self.pod_certificate.to_dict()
        else:
            pod_certificate = self.pod_certificate

        secret: dict[str, Any] | None | Unset
        if isinstance(self.secret, Unset):
            secret = UNSET
        elif isinstance(self.secret, Corev1SecretProjection):
            secret = self.secret.to_dict()
        else:
            secret = self.secret

        service_account_token: dict[str, Any] | None | Unset
        if isinstance(self.service_account_token, Unset):
            service_account_token = UNSET
        elif isinstance(
            self.service_account_token, Corev1ServiceAccountTokenProjection
        ):
            service_account_token = self.service_account_token.to_dict()
        else:
            service_account_token = self.service_account_token

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if cluster_trust_bundle is not UNSET:
            field_dict["clusterTrustBundle"] = cluster_trust_bundle
        if config_map is not UNSET:
            field_dict["configMap"] = config_map
        if downward_api is not UNSET:
            field_dict["downwardAPI"] = downward_api
        if pod_certificate is not UNSET:
            field_dict["podCertificate"] = pod_certificate
        if secret is not UNSET:
            field_dict["secret"] = secret
        if service_account_token is not UNSET:
            field_dict["serviceAccountToken"] = service_account_token

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_cluster_trust_bundle_projection import (
            Corev1ClusterTrustBundleProjection,
        )
        from ..models.corev_1_config_map_projection import Corev1ConfigMapProjection
        from ..models.corev_1_downward_api_projection import Corev1DownwardAPIProjection
        from ..models.corev_1_pod_certificate_projection import (
            Corev1PodCertificateProjection,
        )
        from ..models.corev_1_secret_projection import Corev1SecretProjection
        from ..models.corev_1_service_account_token_projection import (
            Corev1ServiceAccountTokenProjection,
        )

        d = dict(src_dict)

        def _parse_cluster_trust_bundle(
            data: object,
        ) -> Corev1ClusterTrustBundleProjection | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                cluster_trust_bundle_type_1 = (
                    Corev1ClusterTrustBundleProjection.from_dict(data)
                )

                return cluster_trust_bundle_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1ClusterTrustBundleProjection | None | Unset, data)

        cluster_trust_bundle = _parse_cluster_trust_bundle(
            d.pop("clusterTrustBundle", UNSET)
        )

        def _parse_config_map(data: object) -> Corev1ConfigMapProjection | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                config_map_type_1 = Corev1ConfigMapProjection.from_dict(data)

                return config_map_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1ConfigMapProjection | None | Unset, data)

        config_map = _parse_config_map(d.pop("configMap", UNSET))

        def _parse_downward_api(
            data: object,
        ) -> Corev1DownwardAPIProjection | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                downward_api_type_1 = Corev1DownwardAPIProjection.from_dict(data)

                return downward_api_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1DownwardAPIProjection | None | Unset, data)

        downward_api = _parse_downward_api(d.pop("downwardAPI", UNSET))

        def _parse_pod_certificate(
            data: object,
        ) -> Corev1PodCertificateProjection | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                pod_certificate_type_1 = Corev1PodCertificateProjection.from_dict(data)

                return pod_certificate_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1PodCertificateProjection | None | Unset, data)

        pod_certificate = _parse_pod_certificate(d.pop("podCertificate", UNSET))

        def _parse_secret(data: object) -> Corev1SecretProjection | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                secret_type_1 = Corev1SecretProjection.from_dict(data)

                return secret_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1SecretProjection | None | Unset, data)

        secret = _parse_secret(d.pop("secret", UNSET))

        def _parse_service_account_token(
            data: object,
        ) -> Corev1ServiceAccountTokenProjection | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                service_account_token_type_1 = (
                    Corev1ServiceAccountTokenProjection.from_dict(data)
                )

                return service_account_token_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1ServiceAccountTokenProjection | None | Unset, data)

        service_account_token = _parse_service_account_token(
            d.pop("serviceAccountToken", UNSET)
        )

        corev_1_volume_projection = cls(
            cluster_trust_bundle=cluster_trust_bundle,
            config_map=config_map,
            downward_api=downward_api,
            pod_certificate=pod_certificate,
            secret=secret,
            service_account_token=service_account_token,
        )

        corev_1_volume_projection.additional_properties = d
        return corev_1_volume_projection

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
