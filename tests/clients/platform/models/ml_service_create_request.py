from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.backend import Backend
    from ..models.env_var import EnvVar
    from ..models.ml_service_create_request_env_from_item import (
        MLServiceCreateRequestEnvFromItem,
    )
    from ..models.ml_service_create_request_volume_mounts_item import (
        MLServiceCreateRequestVolumeMountsItem,
    )
    from ..models.ml_service_create_request_volumes_item import (
        MLServiceCreateRequestVolumesItem,
    )
    from ..models.ml_service_route import MLServiceRoute
    from ..models.service_port import ServicePort
    from ..models.workload_config_map import WorkloadConfigMap


T = TypeVar("T", bound="MLServiceCreateRequest")


@_attrs_define
class MLServiceCreateRequest:
    r"""
    Example:
        {'backend': {'engine': 'llminference', 'name': 'kserve'}, 'configMaps': [{'data': {'server.yaml': 'maxTokens:
            4096\n'}, 'name': 'llama3-serving-config'}], 'description': 'Llama3-8B online inference service.',
            'displayName': 'Llama3 chat service', 'env': [{'name': 'MAX_TOKENS', 'value': '4096'}], 'envFrom':
            [{'configMapRef': {'name': 'llama3-serving-config'}}], 'image': 'registry.axisml.io/serving/vllm:0.6.0',
            'modelName': 'llama3-8b', 'modelVersion': '1.2.0', 'name': 'llama3-chat', 'poolName': 'gpu-a100', 'ports':
            [{'name': 'http', 'port': 8080}], 'replicas': 3, 'route': {'enabled': True, 'path': '/v1/models/llama3-8b'},
            'unitName': 'a100-1x', 'volumeMounts': [{'mountPath': '/etc/axisml', 'name': 'config', 'readOnly': True}],
            'volumes': [{'configMap': {'name': 'llama3-serving-config'}, 'name': 'config'}]}

    Attributes:
        image (str): Serving container image reference.
        model_name (str): Model artifact definition name to serve.
        model_version (str): Model artifact version to serve.
        name (str): Service name (unique within the tenant).
        pool_name (str): Resource pool to schedule the service onto.
        ports (list[ServicePort]): Container ports exposed by the service (at least one).
        replicas (int): Desired replica count.
        unit_name (str): Resource unit (shape) within the pool.
        args (list[str] | Unset): Container args override.
        backend (Backend | Unset):  Example: {'engine': 'pytorchjob', 'name': 'native'}.
        command (list[str] | Unset): Container entrypoint override.
        config_maps (list[WorkloadConfigMap] | Unset): ConfigMaps created and owned by this MLService.
        description (str | Unset): Free-text service description.
        display_name (str | Unset): Human-readable service label.
        env (list[EnvVar] | Unset): Environment variables injected into the serving pods.
        env_from (list[MLServiceCreateRequestEnvFromItem] | Unset): Environment sources injected into the serving pods
            (pass-through to the K8s EnvFromSource shape, including configMapRef).
        route (MLServiceRoute | Unset):  Example: {'enabled': True, 'path': '/v1/models/llama3-8b'}.
        volume_mounts (list[MLServiceCreateRequestVolumeMountsItem] | Unset): Container volume mounts (pass-through to
            the K8s VolumeMount shape).
        volumes (list[MLServiceCreateRequestVolumesItem] | Unset): Pod volumes (pass-through to the K8s PodSpec volumes
            shape, including configMap).
    """

    image: str
    model_name: str
    model_version: str
    name: str
    pool_name: str
    ports: list[ServicePort]
    replicas: int
    unit_name: str
    args: list[str] | Unset = UNSET
    backend: Backend | Unset = UNSET
    command: list[str] | Unset = UNSET
    config_maps: list[WorkloadConfigMap] | Unset = UNSET
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    env: list[EnvVar] | Unset = UNSET
    env_from: list[MLServiceCreateRequestEnvFromItem] | Unset = UNSET
    route: MLServiceRoute | Unset = UNSET
    volume_mounts: list[MLServiceCreateRequestVolumeMountsItem] | Unset = UNSET
    volumes: list[MLServiceCreateRequestVolumesItem] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        image = self.image

        model_name = self.model_name

        model_version = self.model_version

        name = self.name

        pool_name = self.pool_name

        ports = []
        for ports_item_data in self.ports:
            ports_item = ports_item_data.to_dict()
            ports.append(ports_item)

        replicas = self.replicas

        unit_name = self.unit_name

        args: list[str] | Unset = UNSET
        if not isinstance(self.args, Unset):
            args = self.args

        backend: dict[str, Any] | Unset = UNSET
        if not isinstance(self.backend, Unset):
            backend = self.backend.to_dict()

        command: list[str] | Unset = UNSET
        if not isinstance(self.command, Unset):
            command = self.command

        config_maps: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.config_maps, Unset):
            config_maps = []
            for config_maps_item_data in self.config_maps:
                config_maps_item = config_maps_item_data.to_dict()
                config_maps.append(config_maps_item)

        description = self.description

        display_name = self.display_name

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

        route: dict[str, Any] | Unset = UNSET
        if not isinstance(self.route, Unset):
            route = self.route.to_dict()

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
        field_dict.update(
            {
                "image": image,
                "modelName": model_name,
                "modelVersion": model_version,
                "name": name,
                "poolName": pool_name,
                "ports": ports,
                "replicas": replicas,
                "unitName": unit_name,
            }
        )
        if args is not UNSET:
            field_dict["args"] = args
        if backend is not UNSET:
            field_dict["backend"] = backend
        if command is not UNSET:
            field_dict["command"] = command
        if config_maps is not UNSET:
            field_dict["configMaps"] = config_maps
        if description is not UNSET:
            field_dict["description"] = description
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if env is not UNSET:
            field_dict["env"] = env
        if env_from is not UNSET:
            field_dict["envFrom"] = env_from
        if route is not UNSET:
            field_dict["route"] = route
        if volume_mounts is not UNSET:
            field_dict["volumeMounts"] = volume_mounts
        if volumes is not UNSET:
            field_dict["volumes"] = volumes

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.backend import Backend
        from ..models.env_var import EnvVar
        from ..models.ml_service_create_request_env_from_item import (
            MLServiceCreateRequestEnvFromItem,
        )
        from ..models.ml_service_create_request_volume_mounts_item import (
            MLServiceCreateRequestVolumeMountsItem,
        )
        from ..models.ml_service_create_request_volumes_item import (
            MLServiceCreateRequestVolumesItem,
        )
        from ..models.ml_service_route import MLServiceRoute
        from ..models.service_port import ServicePort
        from ..models.workload_config_map import WorkloadConfigMap

        d = dict(src_dict)
        image = d.pop("image")

        model_name = d.pop("modelName")

        model_version = d.pop("modelVersion")

        name = d.pop("name")

        pool_name = d.pop("poolName")

        ports = []
        _ports = d.pop("ports")
        for ports_item_data in _ports:
            ports_item = ServicePort.from_dict(ports_item_data)

            ports.append(ports_item)

        replicas = d.pop("replicas")

        unit_name = d.pop("unitName")

        args = cast(list[str], d.pop("args", UNSET))

        _backend = d.pop("backend", UNSET)
        backend: Backend | Unset
        if isinstance(_backend, Unset):
            backend = UNSET
        else:
            backend = Backend.from_dict(_backend)

        command = cast(list[str], d.pop("command", UNSET))

        _config_maps = d.pop("configMaps", UNSET)
        config_maps: list[WorkloadConfigMap] | Unset = UNSET
        if _config_maps is not UNSET:
            config_maps = []
            for config_maps_item_data in _config_maps:
                config_maps_item = WorkloadConfigMap.from_dict(config_maps_item_data)

                config_maps.append(config_maps_item)

        description = d.pop("description", UNSET)

        display_name = d.pop("displayName", UNSET)

        _env = d.pop("env", UNSET)
        env: list[EnvVar] | Unset = UNSET
        if _env is not UNSET:
            env = []
            for env_item_data in _env:
                env_item = EnvVar.from_dict(env_item_data)

                env.append(env_item)

        _env_from = d.pop("envFrom", UNSET)
        env_from: list[MLServiceCreateRequestEnvFromItem] | Unset = UNSET
        if _env_from is not UNSET:
            env_from = []
            for env_from_item_data in _env_from:
                env_from_item = MLServiceCreateRequestEnvFromItem.from_dict(
                    env_from_item_data
                )

                env_from.append(env_from_item)

        _route = d.pop("route", UNSET)
        route: MLServiceRoute | Unset
        if isinstance(_route, Unset):
            route = UNSET
        else:
            route = MLServiceRoute.from_dict(_route)

        _volume_mounts = d.pop("volumeMounts", UNSET)
        volume_mounts: list[MLServiceCreateRequestVolumeMountsItem] | Unset = UNSET
        if _volume_mounts is not UNSET:
            volume_mounts = []
            for volume_mounts_item_data in _volume_mounts:
                volume_mounts_item = MLServiceCreateRequestVolumeMountsItem.from_dict(
                    volume_mounts_item_data
                )

                volume_mounts.append(volume_mounts_item)

        _volumes = d.pop("volumes", UNSET)
        volumes: list[MLServiceCreateRequestVolumesItem] | Unset = UNSET
        if _volumes is not UNSET:
            volumes = []
            for volumes_item_data in _volumes:
                volumes_item = MLServiceCreateRequestVolumesItem.from_dict(
                    volumes_item_data
                )

                volumes.append(volumes_item)

        ml_service_create_request = cls(
            image=image,
            model_name=model_name,
            model_version=model_version,
            name=name,
            pool_name=pool_name,
            ports=ports,
            replicas=replicas,
            unit_name=unit_name,
            args=args,
            backend=backend,
            command=command,
            config_maps=config_maps,
            description=description,
            display_name=display_name,
            env=env,
            env_from=env_from,
            route=route,
            volume_mounts=volume_mounts,
            volumes=volumes,
        )

        ml_service_create_request.additional_properties = d
        return ml_service_create_request

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
