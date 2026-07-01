from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.string_map import StringMap


T = TypeVar("T", bound="DataVolumeCreateRequest")


@_attrs_define
class DataVolumeCreateRequest:
    """
    Example:
        {'accessModes': ['ReadWriteMany'], 'description': 'Shared raw datasets directory.', 'name': 'shared-datasets',
            'size': '2Ti', 'storageClass': 'nfs-rwx'}

    Attributes:
        name (str): Data volume name, unique within the tenant.
        size (str): Requested storage size as a Kubernetes quantity (e.g. 200Gi).
        access_modes (list[str] | Unset): Access modes; defaults to [ReadWriteOnce] when empty.
        description (str | Unset): Free-text description of the volume.
        labels (StringMap | Unset):
        storage_class (str | Unset): StorageClass backing the volume; cluster default when empty.
    """

    name: str
    size: str
    access_modes: list[str] | Unset = UNSET
    description: str | Unset = UNSET
    labels: StringMap | Unset = UNSET
    storage_class: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        size = self.size

        access_modes: list[str] | Unset = UNSET
        if not isinstance(self.access_modes, Unset):
            access_modes = self.access_modes

        description = self.description

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        storage_class = self.storage_class

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
                "size": size,
            }
        )
        if access_modes is not UNSET:
            field_dict["accessModes"] = access_modes
        if description is not UNSET:
            field_dict["description"] = description
        if labels is not UNSET:
            field_dict["labels"] = labels
        if storage_class is not UNSET:
            field_dict["storageClass"] = storage_class

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.string_map import StringMap

        d = dict(src_dict)
        name = d.pop("name")

        size = d.pop("size")

        access_modes = cast(list[str], d.pop("accessModes", UNSET))

        description = d.pop("description", UNSET)

        _labels = d.pop("labels", UNSET)
        labels: StringMap | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = StringMap.from_dict(_labels)

        storage_class = d.pop("storageClass", UNSET)

        data_volume_create_request = cls(
            name=name,
            size=size,
            access_modes=access_modes,
            description=description,
            labels=labels,
            storage_class=storage_class,
        )

        data_volume_create_request.additional_properties = d
        return data_volume_create_request

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
