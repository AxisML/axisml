from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.env_var import EnvVar
    from ..models.workspace_lifecycle import WorkspaceLifecycle
    from ..models.workspace_volume import WorkspaceVolume


T = TypeVar("T", bound="WorkspaceCreateRequest")


@_attrs_define
class WorkspaceCreateRequest:
    """
    Example:
        {'containerPort': 8888, 'description': 'JupyterLab interactive development environment.', 'displayName': 'Vision
            team dev environment', 'image': 'registry.axisml.io/dev/jupyter:3.0.0', 'lifecycle': {'idleTimeoutSeconds':
            3600}, 'name': 'notebook-dev', 'poolName': 'gpu-a100', 'unitName': 'a100-1x', 'volumes': [{'mountPath':
            '/home/jovyan/work', 'name': 'notebook-data'}]}

    Attributes:
        image (str): Container image for the dev environment.
        name (str): Workspace name (unique within the tenant).
        pool_name (str): Resource pool to schedule the workspace onto.
        unit_name (str): Resource unit (shape) within the pool.
        args (list[str] | Unset): Container args override.
        command (list[str] | Unset): Container entrypoint override.
        container_port (int | Unset): Port the dev server listens on; defaults from the image when omitted.
        description (str | Unset): Free-text workspace description.
        display_name (str | Unset): Human-readable workspace label.
        env (list[EnvVar] | Unset): Environment variables injected into the container.
        lifecycle (WorkspaceLifecycle | Unset):  Example: {'idleTimeoutSeconds': 3600}.
        volumes (list[WorkspaceVolume] | Unset): Data volumes to mount into the workspace.
    """

    image: str
    name: str
    pool_name: str
    unit_name: str
    args: list[str] | Unset = UNSET
    command: list[str] | Unset = UNSET
    container_port: int | Unset = UNSET
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    env: list[EnvVar] | Unset = UNSET
    lifecycle: WorkspaceLifecycle | Unset = UNSET
    volumes: list[WorkspaceVolume] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        image = self.image

        name = self.name

        pool_name = self.pool_name

        unit_name = self.unit_name

        args: list[str] | Unset = UNSET
        if not isinstance(self.args, Unset):
            args = self.args

        command: list[str] | Unset = UNSET
        if not isinstance(self.command, Unset):
            command = self.command

        container_port = self.container_port

        description = self.description

        display_name = self.display_name

        env: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.env, Unset):
            env = []
            for env_item_data in self.env:
                env_item = env_item_data.to_dict()
                env.append(env_item)

        lifecycle: dict[str, Any] | Unset = UNSET
        if not isinstance(self.lifecycle, Unset):
            lifecycle = self.lifecycle.to_dict()

        volumes: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.volumes, Unset):
            volumes = []
            for volumes_item_data in self.volumes:
                volumes_item = volumes_item_data.to_dict()
                volumes.append(volumes_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "image": image,
                "name": name,
                "poolName": pool_name,
                "unitName": unit_name,
            }
        )
        if args is not UNSET:
            field_dict["args"] = args
        if command is not UNSET:
            field_dict["command"] = command
        if container_port is not UNSET:
            field_dict["containerPort"] = container_port
        if description is not UNSET:
            field_dict["description"] = description
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if env is not UNSET:
            field_dict["env"] = env
        if lifecycle is not UNSET:
            field_dict["lifecycle"] = lifecycle
        if volumes is not UNSET:
            field_dict["volumes"] = volumes

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.env_var import EnvVar
        from ..models.workspace_lifecycle import WorkspaceLifecycle
        from ..models.workspace_volume import WorkspaceVolume

        d = dict(src_dict)
        image = d.pop("image")

        name = d.pop("name")

        pool_name = d.pop("poolName")

        unit_name = d.pop("unitName")

        args = cast(list[str], d.pop("args", UNSET))

        command = cast(list[str], d.pop("command", UNSET))

        container_port = d.pop("containerPort", UNSET)

        description = d.pop("description", UNSET)

        display_name = d.pop("displayName", UNSET)

        _env = d.pop("env", UNSET)
        env: list[EnvVar] | Unset = UNSET
        if _env is not UNSET:
            env = []
            for env_item_data in _env:
                env_item = EnvVar.from_dict(env_item_data)

                env.append(env_item)

        _lifecycle = d.pop("lifecycle", UNSET)
        lifecycle: WorkspaceLifecycle | Unset
        if isinstance(_lifecycle, Unset):
            lifecycle = UNSET
        else:
            lifecycle = WorkspaceLifecycle.from_dict(_lifecycle)

        _volumes = d.pop("volumes", UNSET)
        volumes: list[WorkspaceVolume] | Unset = UNSET
        if _volumes is not UNSET:
            volumes = []
            for volumes_item_data in _volumes:
                volumes_item = WorkspaceVolume.from_dict(volumes_item_data)

                volumes.append(volumes_item)

        workspace_create_request = cls(
            image=image,
            name=name,
            pool_name=pool_name,
            unit_name=unit_name,
            args=args,
            command=command,
            container_port=container_port,
            description=description,
            display_name=display_name,
            env=env,
            lifecycle=lifecycle,
            volumes=volumes,
        )

        workspace_create_request.additional_properties = d
        return workspace_create_request

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
