from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.env_var import EnvVar
    from ..models.resource_map import ResourceMap
    from ..models.role_template_env_from_item import RoleTemplateEnvFromItem
    from ..models.role_template_ports_item import RoleTemplatePortsItem
    from ..models.role_template_volume_mounts_item import RoleTemplateVolumeMountsItem
    from ..models.role_template_volumes_item import RoleTemplateVolumesItem


T = TypeVar("T", bound="RoleTemplate")


@_attrs_define
class RoleTemplate:
    """
    Example:
        {'args': ['--epochs', '90', '--batch-size', '256'], 'command': ['python', 'train.py'], 'env': [{'name':
            'NCCL_DEBUG', 'value': 'INFO'}], 'envFrom': [{'configMapRef': {'name': 'resnet-training-config'}}], 'image':
            'registry.axisml.io/training/resnet:1.4.0', 'ports': [{'containerPort': 8080, 'name': 'http', 'protocol':
            'TCP'}], 'resources': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}, 'volumeMounts': [{'mountPath':
            '/data', 'name': 'data'}, {'mountPath': '/etc/axisml', 'name': 'config', 'readOnly': True}], 'volumes':
            [{'name': 'data', 'persistentVolumeClaim': {'claimName': 'resnet-imagenet'}}, {'configMap': {'name': 'resnet-
            training-config'}, 'name': 'config'}]}

    Attributes:
        args (list[str] | Unset): Container args override.
        command (list[str] | Unset): Container entrypoint override.
        env (list[EnvVar] | Unset): Environment variables injected into the role's pods.
        env_from (list[RoleTemplateEnvFromItem] | Unset): Environment sources injected into the role's pods (pass-
            through to the K8s EnvFromSource shape, including configMapRef).
        image (str | Unset): Container image reference for this role's pods.
        ports (list[RoleTemplatePortsItem] | Unset): Container ports (pass-through to the K8s container ports shape).
        resources (ResourceMap | Unset): Kubernetes-style resource quantity map (e.g., {"cpu": "100", "memory": "1Ti",
            "nvidia.com/gpu": "8"}).
        volume_mounts (list[RoleTemplateVolumeMountsItem] | Unset): Container volume mounts (pass-through to the K8s
            shape).
        volumes (list[RoleTemplateVolumesItem] | Unset): Pod volumes (pass-through to the K8s PodSpec volumes shape).
    """

    args: list[str] | Unset = UNSET
    command: list[str] | Unset = UNSET
    env: list[EnvVar] | Unset = UNSET
    env_from: list[RoleTemplateEnvFromItem] | Unset = UNSET
    image: str | Unset = UNSET
    ports: list[RoleTemplatePortsItem] | Unset = UNSET
    resources: ResourceMap | Unset = UNSET
    volume_mounts: list[RoleTemplateVolumeMountsItem] | Unset = UNSET
    volumes: list[RoleTemplateVolumesItem] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        args: list[str] | Unset = UNSET
        if not isinstance(self.args, Unset):
            args = self.args

        command: list[str] | Unset = UNSET
        if not isinstance(self.command, Unset):
            command = self.command

        env: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.env, Unset):
            env = []
            for env_item_data in self.env:
                env_item = env_item_data.to_dict()
                env.append(env_item)

        env_from: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.env_from, Unset):
            env_from = []
            for env_from_item_data in self.env_from:
                env_from_item = env_from_item_data.to_dict()
                env_from.append(env_from_item)

        image = self.image

        ports: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.ports, Unset):
            ports = []
            for ports_item_data in self.ports:
                ports_item = ports_item_data.to_dict()
                ports.append(ports_item)

        resources: dict[str, Any] | Unset = UNSET
        if not isinstance(self.resources, Unset):
            resources = self.resources.to_dict()

        volume_mounts: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.volume_mounts, Unset):
            volume_mounts = []
            for volume_mounts_item_data in self.volume_mounts:
                volume_mounts_item = volume_mounts_item_data.to_dict()
                volume_mounts.append(volume_mounts_item)

        volumes: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.volumes, Unset):
            volumes = []
            for volumes_item_data in self.volumes:
                volumes_item = volumes_item_data.to_dict()
                volumes.append(volumes_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if args is not UNSET:
            field_dict["args"] = args
        if command is not UNSET:
            field_dict["command"] = command
        if env is not UNSET:
            field_dict["env"] = env
        if env_from is not UNSET:
            field_dict["envFrom"] = env_from
        if image is not UNSET:
            field_dict["image"] = image
        if ports is not UNSET:
            field_dict["ports"] = ports
        if resources is not UNSET:
            field_dict["resources"] = resources
        if volume_mounts is not UNSET:
            field_dict["volumeMounts"] = volume_mounts
        if volumes is not UNSET:
            field_dict["volumes"] = volumes

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.env_var import EnvVar
        from ..models.resource_map import ResourceMap
        from ..models.role_template_env_from_item import RoleTemplateEnvFromItem
        from ..models.role_template_ports_item import RoleTemplatePortsItem
        from ..models.role_template_volume_mounts_item import (
            RoleTemplateVolumeMountsItem,
        )
        from ..models.role_template_volumes_item import RoleTemplateVolumesItem

        d = dict(src_dict)
        args = cast(list[str], d.pop("args", UNSET))

        command = cast(list[str], d.pop("command", UNSET))

        _env = d.pop("env", UNSET)
        env: list[EnvVar] | Unset = UNSET
        if _env is not UNSET:
            env = []
            for env_item_data in _env:
                env_item = EnvVar.from_dict(env_item_data)

                env.append(env_item)

        _env_from = d.pop("envFrom", UNSET)
        env_from: list[RoleTemplateEnvFromItem] | Unset = UNSET
        if _env_from is not UNSET:
            env_from = []
            for env_from_item_data in _env_from:
                env_from_item = RoleTemplateEnvFromItem.from_dict(env_from_item_data)

                env_from.append(env_from_item)

        image = d.pop("image", UNSET)

        _ports = d.pop("ports", UNSET)
        ports: list[RoleTemplatePortsItem] | Unset = UNSET
        if _ports is not UNSET:
            ports = []
            for ports_item_data in _ports:
                ports_item = RoleTemplatePortsItem.from_dict(ports_item_data)

                ports.append(ports_item)

        _resources = d.pop("resources", UNSET)
        resources: ResourceMap | Unset
        if isinstance(_resources, Unset):
            resources = UNSET
        else:
            resources = ResourceMap.from_dict(_resources)

        _volume_mounts = d.pop("volumeMounts", UNSET)
        volume_mounts: list[RoleTemplateVolumeMountsItem] | Unset = UNSET
        if _volume_mounts is not UNSET:
            volume_mounts = []
            for volume_mounts_item_data in _volume_mounts:
                volume_mounts_item = RoleTemplateVolumeMountsItem.from_dict(
                    volume_mounts_item_data
                )

                volume_mounts.append(volume_mounts_item)

        _volumes = d.pop("volumes", UNSET)
        volumes: list[RoleTemplateVolumesItem] | Unset = UNSET
        if _volumes is not UNSET:
            volumes = []
            for volumes_item_data in _volumes:
                volumes_item = RoleTemplateVolumesItem.from_dict(volumes_item_data)

                volumes.append(volumes_item)

        role_template = cls(
            args=args,
            command=command,
            env=env,
            env_from=env_from,
            image=image,
            ports=ports,
            resources=resources,
            volume_mounts=volume_mounts,
            volumes=volumes,
        )

        role_template.additional_properties = d
        return role_template

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
