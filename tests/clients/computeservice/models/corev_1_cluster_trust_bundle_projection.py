from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.metav_1_label_selector import Metav1LabelSelector


T = TypeVar("T", bound="Corev1ClusterTrustBundleProjection")


@_attrs_define
class Corev1ClusterTrustBundleProjection:
    """
    Attributes:
        path (str):
        label_selector (Metav1LabelSelector | None | Unset):
        name (None | str | Unset):
        optional (bool | None | Unset):
        signer_name (None | str | Unset):
    """

    path: str
    label_selector: Metav1LabelSelector | None | Unset = UNSET
    name: None | str | Unset = UNSET
    optional: bool | None | Unset = UNSET
    signer_name: None | str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.metav_1_label_selector import Metav1LabelSelector

        path = self.path

        label_selector: dict[str, Any] | None | Unset
        if isinstance(self.label_selector, Unset):
            label_selector = UNSET
        elif isinstance(self.label_selector, Metav1LabelSelector):
            label_selector = self.label_selector.to_dict()
        else:
            label_selector = self.label_selector

        name: None | str | Unset
        if isinstance(self.name, Unset):
            name = UNSET
        else:
            name = self.name

        optional: bool | None | Unset
        if isinstance(self.optional, Unset):
            optional = UNSET
        else:
            optional = self.optional

        signer_name: None | str | Unset
        if isinstance(self.signer_name, Unset):
            signer_name = UNSET
        else:
            signer_name = self.signer_name

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "path": path,
            }
        )
        if label_selector is not UNSET:
            field_dict["labelSelector"] = label_selector
        if name is not UNSET:
            field_dict["name"] = name
        if optional is not UNSET:
            field_dict["optional"] = optional
        if signer_name is not UNSET:
            field_dict["signerName"] = signer_name

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.metav_1_label_selector import Metav1LabelSelector

        d = dict(src_dict)
        path = d.pop("path")

        def _parse_label_selector(data: object) -> Metav1LabelSelector | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                label_selector_type_1 = Metav1LabelSelector.from_dict(data)

                return label_selector_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Metav1LabelSelector | None | Unset, data)

        label_selector = _parse_label_selector(d.pop("labelSelector", UNSET))

        def _parse_name(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        name = _parse_name(d.pop("name", UNSET))

        def _parse_optional(data: object) -> bool | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(bool | None | Unset, data)

        optional = _parse_optional(d.pop("optional", UNSET))

        def _parse_signer_name(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        signer_name = _parse_signer_name(d.pop("signerName", UNSET))

        corev_1_cluster_trust_bundle_projection = cls(
            path=path,
            label_selector=label_selector,
            name=name,
            optional=optional,
            signer_name=signer_name,
        )

        corev_1_cluster_trust_bundle_projection.additional_properties = d
        return corev_1_cluster_trust_bundle_projection

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
