from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.create_volume_request_labels import CreateVolumeRequestLabels


T = TypeVar("T", bound="CreateVolumeRequest")


@_attrs_define
class CreateVolumeRequest:
    """
    Example:
        {'accessModes': ['ReadWriteMany'], 'description': 'Shared raw datasets directory', 'name': 'shared-datasets',
            'namespace': 'team-vision', 'size': '2Ti', 'storageClass': 'nfs-rwx'}

    Attributes:
        access_modes (list[str] | Unset): Access modes; defaults to [ReadWriteOnce] when empty.
        description (str | Unset): Free-text description of the volume.
        labels (CreateVolumeRequestLabels | Unset): User-defined labels to set on the volume.
        name (str | Unset): Deterministic volume name supplied by the caller.
        namespace (str | Unset): Physical Kubernetes namespace to materialise the volume in.
        size (str | Unset): Requested storage size as a Kubernetes quantity (e.g. 50Gi).
        storage_class (str | Unset): StorageClass to back the volume; cluster default when empty.
    """

    access_modes: list[str] | Unset = UNSET
    description: str | Unset = UNSET
    labels: CreateVolumeRequestLabels | Unset = UNSET
    name: str | Unset = UNSET
    namespace: str | Unset = UNSET
    size: str | Unset = UNSET
    storage_class: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        access_modes: list[str] | Unset = UNSET
        if not isinstance(self.access_modes, Unset):
            access_modes = self.access_modes

        description = self.description

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        name = self.name

        namespace = self.namespace

        size = self.size

        storage_class = self.storage_class

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if access_modes is not UNSET:
            field_dict["accessModes"] = access_modes
        if description is not UNSET:
            field_dict["description"] = description
        if labels is not UNSET:
            field_dict["labels"] = labels
        if name is not UNSET:
            field_dict["name"] = name
        if namespace is not UNSET:
            field_dict["namespace"] = namespace
        if size is not UNSET:
            field_dict["size"] = size
        if storage_class is not UNSET:
            field_dict["storageClass"] = storage_class

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.create_volume_request_labels import CreateVolumeRequestLabels

        d = dict(src_dict)
        access_modes = cast(list[str], d.pop("accessModes", UNSET))

        description = d.pop("description", UNSET)

        _labels = d.pop("labels", UNSET)
        labels: CreateVolumeRequestLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = CreateVolumeRequestLabels.from_dict(_labels)

        name = d.pop("name", UNSET)

        namespace = d.pop("namespace", UNSET)

        size = d.pop("size", UNSET)

        storage_class = d.pop("storageClass", UNSET)

        create_volume_request = cls(
            access_modes=access_modes,
            description=description,
            labels=labels,
            name=name,
            namespace=namespace,
            size=size,
            storage_class=storage_class,
        )

        create_volume_request.additional_properties = d
        return create_volume_request

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
