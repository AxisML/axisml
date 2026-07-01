from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_object_field_selector import Corev1ObjectFieldSelector
    from ..models.corev_1_resource_field_selector import Corev1ResourceFieldSelector


T = TypeVar("T", bound="Corev1DownwardAPIVolumeFile")


@_attrs_define
class Corev1DownwardAPIVolumeFile:
    """
    Attributes:
        path (str):
        field_ref (Corev1ObjectFieldSelector | None | Unset):
        mode (int | None | Unset):
        resource_field_ref (Corev1ResourceFieldSelector | None | Unset):
    """

    path: str
    field_ref: Corev1ObjectFieldSelector | None | Unset = UNSET
    mode: int | None | Unset = UNSET
    resource_field_ref: Corev1ResourceFieldSelector | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.corev_1_object_field_selector import Corev1ObjectFieldSelector
        from ..models.corev_1_resource_field_selector import Corev1ResourceFieldSelector

        path = self.path

        field_ref: dict[str, Any] | None | Unset
        if isinstance(self.field_ref, Unset):
            field_ref = UNSET
        elif isinstance(self.field_ref, Corev1ObjectFieldSelector):
            field_ref = self.field_ref.to_dict()
        else:
            field_ref = self.field_ref

        mode: int | None | Unset
        if isinstance(self.mode, Unset):
            mode = UNSET
        else:
            mode = self.mode

        resource_field_ref: dict[str, Any] | None | Unset
        if isinstance(self.resource_field_ref, Unset):
            resource_field_ref = UNSET
        elif isinstance(self.resource_field_ref, Corev1ResourceFieldSelector):
            resource_field_ref = self.resource_field_ref.to_dict()
        else:
            resource_field_ref = self.resource_field_ref

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "path": path,
            }
        )
        if field_ref is not UNSET:
            field_dict["fieldRef"] = field_ref
        if mode is not UNSET:
            field_dict["mode"] = mode
        if resource_field_ref is not UNSET:
            field_dict["resourceFieldRef"] = resource_field_ref

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_object_field_selector import Corev1ObjectFieldSelector
        from ..models.corev_1_resource_field_selector import Corev1ResourceFieldSelector

        d = dict(src_dict)
        path = d.pop("path")

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

        def _parse_mode(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        mode = _parse_mode(d.pop("mode", UNSET))

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

        corev_1_downward_api_volume_file = cls(
            path=path,
            field_ref=field_ref,
            mode=mode,
            resource_field_ref=resource_field_ref,
        )

        corev_1_downward_api_volume_file.additional_properties = d
        return corev_1_downward_api_volume_file

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
