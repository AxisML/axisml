from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.server_volume_status import ServerVolumeStatus
    from ..models.volume_labels import VolumeLabels


T = TypeVar("T", bound="Volume")


@_attrs_define
class Volume:
    """
    Example:
        {'accessModes': ['ReadWriteMany'], 'createdAt': '2026-02-11T08:00:00Z', 'description': 'Shared raw datasets
            directory', 'name': 'shared-datasets', 'namespace': 'team-vision', 'size': '2Ti', 'status': {'boundCapacity':
            '2Ti', 'mounts': [{'kind': 'Deployment', 'mountPath': '/data/shared', 'running': True, 'workload': 'ws-
            jupyter-3'}], 'phase': 'Bound'}, 'storageClass': 'nfs-rwx'}

    Attributes:
        name (str): Volume (PersistentVolumeClaim) name.
        namespace (str): Physical Kubernetes namespace holding the volume.
        access_modes (list[str] | Unset): Access modes (ReadWriteOnce/ReadWriteMany/ReadOnlyMany). Immutable after
            creation.
        created_at (datetime.datetime | Unset): Volume creation timestamp (RFC3339).
        description (str | Unset): Free-text description of the volume.
        labels (VolumeLabels | Unset): User-defined labels on the volume.
        size (str | Unset): Requested storage size as a Kubernetes quantity (e.g. 50Gi).
        status (None | ServerVolumeStatus | Unset): Live status read from the PVC and pod scan; populated on get/list.
        storage_class (str | Unset): StorageClass backing the volume; cluster default when empty. Immutable after
            creation.
    """

    name: str
    namespace: str
    access_modes: list[str] | Unset = UNSET
    created_at: datetime.datetime | Unset = UNSET
    description: str | Unset = UNSET
    labels: VolumeLabels | Unset = UNSET
    size: str | Unset = UNSET
    status: None | ServerVolumeStatus | Unset = UNSET
    storage_class: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.server_volume_status import ServerVolumeStatus

        name = self.name

        namespace = self.namespace

        access_modes: list[str] | Unset = UNSET
        if not isinstance(self.access_modes, Unset):
            access_modes = self.access_modes

        created_at: str | Unset = UNSET
        if not isinstance(self.created_at, Unset):
            created_at = self.created_at.isoformat()

        description = self.description

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        size = self.size

        status: dict[str, Any] | None | Unset
        if isinstance(self.status, Unset):
            status = UNSET
        elif isinstance(self.status, ServerVolumeStatus):
            status = self.status.to_dict()
        else:
            status = self.status

        storage_class = self.storage_class

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
                "namespace": namespace,
            }
        )
        if access_modes is not UNSET:
            field_dict["accessModes"] = access_modes
        if created_at is not UNSET:
            field_dict["createdAt"] = created_at
        if description is not UNSET:
            field_dict["description"] = description
        if labels is not UNSET:
            field_dict["labels"] = labels
        if size is not UNSET:
            field_dict["size"] = size
        if status is not UNSET:
            field_dict["status"] = status
        if storage_class is not UNSET:
            field_dict["storageClass"] = storage_class

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.server_volume_status import ServerVolumeStatus
        from ..models.volume_labels import VolumeLabels

        d = dict(src_dict)
        name = d.pop("name")

        namespace = d.pop("namespace")

        access_modes = cast(list[str], d.pop("accessModes", UNSET))

        _created_at = d.pop("createdAt", UNSET)
        created_at: datetime.datetime | Unset
        if isinstance(_created_at, Unset):
            created_at = UNSET
        else:
            created_at = datetime.datetime.fromisoformat(_created_at)

        description = d.pop("description", UNSET)

        _labels = d.pop("labels", UNSET)
        labels: VolumeLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = VolumeLabels.from_dict(_labels)

        size = d.pop("size", UNSET)

        def _parse_status(data: object) -> None | ServerVolumeStatus | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                status_type_1 = ServerVolumeStatus.from_dict(data)

                return status_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | ServerVolumeStatus | Unset, data)

        status = _parse_status(d.pop("status", UNSET))

        storage_class = d.pop("storageClass", UNSET)

        volume = cls(
            name=name,
            namespace=namespace,
            access_modes=access_modes,
            created_at=created_at,
            description=description,
            labels=labels,
            size=size,
            status=status,
            storage_class=storage_class,
        )

        volume.additional_properties = d
        return volume

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
