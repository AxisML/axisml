from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.backend import Backend
    from ..models.env_var import EnvVar
    from ..models.ml_service_route import MLServiceRoute
    from ..models.service_port import ServicePort


T = TypeVar("T", bound="MLServiceCreateRequest")


@_attrs_define
class MLServiceCreateRequest:
    """
    Example:
        {'backend': {'engine': 'llminference', 'name': 'kserve'}, 'description': 'Llama3-8B online inference service.',
            'displayName': 'Llama3 chat service', 'env': [{'name': 'MAX_TOKENS', 'value': '4096'}], 'image':
            'registry.axisml.io/serving/vllm:0.6.0', 'modelName': 'llama3-8b', 'modelVersion': '1.2.0', 'name':
            'llama3-chat', 'poolName': 'gpu-a100', 'ports': [{'name': 'http', 'port': 8080}], 'replicas': 3, 'route':
            {'enabled': True, 'path': '/v1/models/llama3-8b'}, 'unitName': 'a100-1x'}

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
        description (str | Unset): Free-text service description.
        display_name (str | Unset): Human-readable service label.
        env (list[EnvVar] | Unset): Environment variables injected into the serving pods.
        route (MLServiceRoute | Unset):  Example: {'enabled': True, 'path': '/v1/models/llama3-8b'}.
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
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    env: list[EnvVar] | Unset = UNSET
    route: MLServiceRoute | Unset = UNSET
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

        description = self.description

        display_name = self.display_name

        env: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.env, Unset):
            env = []
            for env_item_data in self.env:
                env_item = env_item_data.to_dict()
                env.append(env_item)

        route: dict[str, Any] | Unset = UNSET
        if not isinstance(self.route, Unset):
            route = self.route.to_dict()

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
        if description is not UNSET:
            field_dict["description"] = description
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if env is not UNSET:
            field_dict["env"] = env
        if route is not UNSET:
            field_dict["route"] = route

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.backend import Backend
        from ..models.env_var import EnvVar
        from ..models.ml_service_route import MLServiceRoute
        from ..models.service_port import ServicePort

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

        description = d.pop("description", UNSET)

        display_name = d.pop("displayName", UNSET)

        _env = d.pop("env", UNSET)
        env: list[EnvVar] | Unset = UNSET
        if _env is not UNSET:
            env = []
            for env_item_data in _env:
                env_item = EnvVar.from_dict(env_item_data)

                env.append(env_item)

        _route = d.pop("route", UNSET)
        route: MLServiceRoute | Unset
        if isinstance(_route, Unset):
            route = UNSET
        else:
            route = MLServiceRoute.from_dict(_route)

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
            description=description,
            display_name=display_name,
            env=env,
            route=route,
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
