from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="ServerVolumeMount")


@_attrs_define
class ServerVolumeMount:
    """
    Attributes:
        running (bool): Whether the mounting pod is currently running.
        workload (str): Controlling workload (or pod) name.
        kind (str | Unset): Kubernetes controller kind (Deployment/StatefulSet/Job/Pod).
        mount_path (str | Unset): Mount path inside the pod.
    """

    running: bool
    workload: str
    kind: str | Unset = UNSET
    mount_path: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        running = self.running

        workload = self.workload

        kind = self.kind

        mount_path = self.mount_path

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "running": running,
                "workload": workload,
            }
        )
        if kind is not UNSET:
            field_dict["kind"] = kind
        if mount_path is not UNSET:
            field_dict["mountPath"] = mount_path

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        running = d.pop("running")

        workload = d.pop("workload")

        kind = d.pop("kind", UNSET)

        mount_path = d.pop("mountPath", UNSET)

        server_volume_mount = cls(
            running=running,
            workload=workload,
            kind=kind,
            mount_path=mount_path,
        )

        server_volume_mount.additional_properties = d
        return server_volume_mount

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
