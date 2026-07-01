from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_typed_local_object_reference import (
        Corev1TypedLocalObjectReference,
    )
    from ..models.corev_1_typed_object_reference import Corev1TypedObjectReference
    from ..models.corev_1_volume_resource_requirements import (
        Corev1VolumeResourceRequirements,
    )
    from ..models.metav_1_label_selector import Metav1LabelSelector


T = TypeVar("T", bound="Corev1PersistentVolumeClaimSpec")


@_attrs_define
class Corev1PersistentVolumeClaimSpec:
    """
    Attributes:
        access_modes (list[str] | Unset):
        data_source (Corev1TypedLocalObjectReference | None | Unset):
        data_source_ref (Corev1TypedObjectReference | None | Unset):
        resources (Corev1VolumeResourceRequirements | Unset):
        selector (Metav1LabelSelector | None | Unset):
        storage_class_name (None | str | Unset):
        volume_attributes_class_name (None | str | Unset):
        volume_mode (None | str | Unset):
        volume_name (str | Unset):
    """

    access_modes: list[str] | Unset = UNSET
    data_source: Corev1TypedLocalObjectReference | None | Unset = UNSET
    data_source_ref: Corev1TypedObjectReference | None | Unset = UNSET
    resources: Corev1VolumeResourceRequirements | Unset = UNSET
    selector: Metav1LabelSelector | None | Unset = UNSET
    storage_class_name: None | str | Unset = UNSET
    volume_attributes_class_name: None | str | Unset = UNSET
    volume_mode: None | str | Unset = UNSET
    volume_name: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.corev_1_typed_local_object_reference import (
            Corev1TypedLocalObjectReference,
        )
        from ..models.corev_1_typed_object_reference import Corev1TypedObjectReference
        from ..models.metav_1_label_selector import Metav1LabelSelector

        access_modes: list[str] | Unset = UNSET
        if not isinstance(self.access_modes, Unset):
            access_modes = self.access_modes

        data_source: dict[str, Any] | None | Unset
        if isinstance(self.data_source, Unset):
            data_source = UNSET
        elif isinstance(self.data_source, Corev1TypedLocalObjectReference):
            data_source = self.data_source.to_dict()
        else:
            data_source = self.data_source

        data_source_ref: dict[str, Any] | None | Unset
        if isinstance(self.data_source_ref, Unset):
            data_source_ref = UNSET
        elif isinstance(self.data_source_ref, Corev1TypedObjectReference):
            data_source_ref = self.data_source_ref.to_dict()
        else:
            data_source_ref = self.data_source_ref

        resources: dict[str, Any] | Unset = UNSET
        if not isinstance(self.resources, Unset):
            resources = self.resources.to_dict()

        selector: dict[str, Any] | None | Unset
        if isinstance(self.selector, Unset):
            selector = UNSET
        elif isinstance(self.selector, Metav1LabelSelector):
            selector = self.selector.to_dict()
        else:
            selector = self.selector

        storage_class_name: None | str | Unset
        if isinstance(self.storage_class_name, Unset):
            storage_class_name = UNSET
        else:
            storage_class_name = self.storage_class_name

        volume_attributes_class_name: None | str | Unset
        if isinstance(self.volume_attributes_class_name, Unset):
            volume_attributes_class_name = UNSET
        else:
            volume_attributes_class_name = self.volume_attributes_class_name

        volume_mode: None | str | Unset
        if isinstance(self.volume_mode, Unset):
            volume_mode = UNSET
        else:
            volume_mode = self.volume_mode

        volume_name = self.volume_name

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if access_modes is not UNSET:
            field_dict["accessModes"] = access_modes
        if data_source is not UNSET:
            field_dict["dataSource"] = data_source
        if data_source_ref is not UNSET:
            field_dict["dataSourceRef"] = data_source_ref
        if resources is not UNSET:
            field_dict["resources"] = resources
        if selector is not UNSET:
            field_dict["selector"] = selector
        if storage_class_name is not UNSET:
            field_dict["storageClassName"] = storage_class_name
        if volume_attributes_class_name is not UNSET:
            field_dict["volumeAttributesClassName"] = volume_attributes_class_name
        if volume_mode is not UNSET:
            field_dict["volumeMode"] = volume_mode
        if volume_name is not UNSET:
            field_dict["volumeName"] = volume_name

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_typed_local_object_reference import (
            Corev1TypedLocalObjectReference,
        )
        from ..models.corev_1_typed_object_reference import Corev1TypedObjectReference
        from ..models.corev_1_volume_resource_requirements import (
            Corev1VolumeResourceRequirements,
        )
        from ..models.metav_1_label_selector import Metav1LabelSelector

        d = dict(src_dict)
        access_modes = cast(list[str], d.pop("accessModes", UNSET))

        def _parse_data_source(
            data: object,
        ) -> Corev1TypedLocalObjectReference | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                data_source_type_1 = Corev1TypedLocalObjectReference.from_dict(data)

                return data_source_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1TypedLocalObjectReference | None | Unset, data)

        data_source = _parse_data_source(d.pop("dataSource", UNSET))

        def _parse_data_source_ref(
            data: object,
        ) -> Corev1TypedObjectReference | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                data_source_ref_type_1 = Corev1TypedObjectReference.from_dict(data)

                return data_source_ref_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1TypedObjectReference | None | Unset, data)

        data_source_ref = _parse_data_source_ref(d.pop("dataSourceRef", UNSET))

        _resources = d.pop("resources", UNSET)
        resources: Corev1VolumeResourceRequirements | Unset
        if isinstance(_resources, Unset):
            resources = UNSET
        else:
            resources = Corev1VolumeResourceRequirements.from_dict(_resources)

        def _parse_selector(data: object) -> Metav1LabelSelector | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                selector_type_1 = Metav1LabelSelector.from_dict(data)

                return selector_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Metav1LabelSelector | None | Unset, data)

        selector = _parse_selector(d.pop("selector", UNSET))

        def _parse_storage_class_name(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        storage_class_name = _parse_storage_class_name(d.pop("storageClassName", UNSET))

        def _parse_volume_attributes_class_name(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        volume_attributes_class_name = _parse_volume_attributes_class_name(
            d.pop("volumeAttributesClassName", UNSET)
        )

        def _parse_volume_mode(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        volume_mode = _parse_volume_mode(d.pop("volumeMode", UNSET))

        volume_name = d.pop("volumeName", UNSET)

        corev_1_persistent_volume_claim_spec = cls(
            access_modes=access_modes,
            data_source=data_source,
            data_source_ref=data_source_ref,
            resources=resources,
            selector=selector,
            storage_class_name=storage_class_name,
            volume_attributes_class_name=volume_attributes_class_name,
            volume_mode=volume_mode,
            volume_name=volume_name,
        )

        corev_1_persistent_volume_claim_spec.additional_properties = d
        return corev_1_persistent_volume_claim_spec

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
