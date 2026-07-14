from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.ml_service_desired_state import MLServiceDesiredState
from ..models.ml_service_phase import MLServicePhase
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.backend import Backend
    from ..models.env_var import EnvVar
    from ..models.ml_service_route import MLServiceRoute
    from ..models.resource_map import ResourceMap
    from ..models.service_port import ServicePort


T = TypeVar("T", bound="MLService")


@_attrs_define
class MLService:
    """
    Example:
        {'accessUrl': 'https://gateway.axisml.io/v1/models/llama3-8b', 'args': ['--model', 'meta-llama/Llama-3-8b', '--
            max-model-len', '8192'], 'backend': {'engine': 'llminference', 'name': 'kserve'}, 'command': ['python', '-m',
            'vllm.entrypoints.openai.api_server'], 'computeNamespace': 'axisml-team-nlp', 'createdAt':
            '2026-06-20T08:00:00Z', 'description': 'Llama3-8B online inference service.', 'desiredState': 'Running',
            'displayName': 'Llama3 chat service', 'env': [{'name': 'MAX_TOKENS', 'value': '4096'}], 'id':
            '5d2c9b41-3e8f-4a1c-9d7e-6b4f2a1c8e90', 'image': 'registry.axisml.io/serving/vllm:0.6.0', 'message': 'All
            replicas ready.', 'modelName': 'llama3-8b', 'modelVersion': '1.2.0', 'name': 'llama3-chat', 'namespace': 'team-
            nlp', 'owner': 'zhang.san', 'ownerId': '9f8e7d6c-5b4a-3210-fedc-ba9876543210', 'phase': 'Ready', 'poolName':
            'gpu-a100', 'ports': [{'name': 'http', 'port': 8080}], 'readyReplicas': 3, 'replicas': 3, 'resources': {'cpu':
            '8', 'memory': '64Gi', 'nvidia.com/gpu': '1'}, 'route': {'enabled': True, 'path': '/v1/models/llama3-8b'},
            'tenantDisplayName': 'Vision Team', 'tenantName': 'team-nlp', 'unitName': 'a100-1x', 'updatedAt':
            '2026-06-28T09:30:00Z'}

    Attributes:
        created_at (datetime.datetime): Time the service was created.
        id (UUID): Stable service identifier.
        name (str): Service name (unique within the tenant).
        namespace (str): Platform tenant namespace the service belongs to.
        owner (str): Username of the service owner.
        tenant_name (str): Tenant identifier owning the service.
        updated_at (datetime.datetime): Time the service was last updated.
        access_url (str | Unset): Resolved URL clients use to reach the service.
        args (list[str] | Unset): Container args override.
        backend (Backend | Unset):  Example: {'engine': 'pytorchjob', 'name': 'native'}.
        command (list[str] | Unset): Container entrypoint override.
        compute_namespace (str | Unset): Underlying compute (Kubernetes) namespace running the service.
        description (str | Unset): Free-text service description.
        desired_state (MLServiceDesiredState | Unset):
        display_name (str | Unset): Human-readable service label.
        env (list[EnvVar] | Unset): Environment variables injected into the serving pods.
        image (str | Unset): Serving container image reference.
        message (str | Unset): Human-readable status detail for the current phase.
        model_name (str | Unset): Model artifact definition name being served.
        model_version (str | Unset): Model artifact version being served.
        owner_id (UUID | Unset): User ID of the service owner.
        phase (MLServicePhase | Unset):
        pool_name (str | Unset): Resource pool the service is scheduled onto.
        ports (list[ServicePort] | Unset): Container ports exposed by the service.
        ready_replicas (int | Unset): Replicas that have passed readiness.
        replicas (int | Unset): Desired replica count.
        resources (ResourceMap | Unset): Kubernetes-style resource quantity map (e.g., {"cpu": "100", "memory": "1Ti",
            "nvidia.com/gpu": "8"}).
        route (MLServiceRoute | Unset):  Example: {'enabled': True, 'path': '/v1/models/llama3-8b'}.
        tenant_display_name (str | Unset): Human-readable tenant name.
        unit_name (str | Unset): Resource unit (shape) within the pool.
    """

    created_at: datetime.datetime
    id: UUID
    name: str
    namespace: str
    owner: str
    tenant_name: str
    updated_at: datetime.datetime
    access_url: str | Unset = UNSET
    args: list[str] | Unset = UNSET
    backend: Backend | Unset = UNSET
    command: list[str] | Unset = UNSET
    compute_namespace: str | Unset = UNSET
    description: str | Unset = UNSET
    desired_state: MLServiceDesiredState | Unset = UNSET
    display_name: str | Unset = UNSET
    env: list[EnvVar] | Unset = UNSET
    image: str | Unset = UNSET
    message: str | Unset = UNSET
    model_name: str | Unset = UNSET
    model_version: str | Unset = UNSET
    owner_id: UUID | Unset = UNSET
    phase: MLServicePhase | Unset = UNSET
    pool_name: str | Unset = UNSET
    ports: list[ServicePort] | Unset = UNSET
    ready_replicas: int | Unset = UNSET
    replicas: int | Unset = UNSET
    resources: ResourceMap | Unset = UNSET
    route: MLServiceRoute | Unset = UNSET
    tenant_display_name: str | Unset = UNSET
    unit_name: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        id = str(self.id)

        name = self.name

        namespace = self.namespace

        owner = self.owner

        tenant_name = self.tenant_name

        updated_at = self.updated_at.isoformat()

        access_url = self.access_url

        args: list[str] | Unset = UNSET
        if not isinstance(self.args, Unset):
            args = self.args

        backend: dict[str, Any] | Unset = UNSET
        if not isinstance(self.backend, Unset):
            backend = self.backend.to_dict()

        command: list[str] | Unset = UNSET
        if not isinstance(self.command, Unset):
            command = self.command

        compute_namespace = self.compute_namespace

        description = self.description

        desired_state: str | Unset = UNSET
        if not isinstance(self.desired_state, Unset):
            desired_state = self.desired_state.value

        display_name = self.display_name

        env: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.env, Unset):
            env = []
            for env_item_data in self.env:
                env_item = env_item_data.to_dict()
                env.append(env_item)

        image = self.image

        message = self.message

        model_name = self.model_name

        model_version = self.model_version

        owner_id: str | Unset = UNSET
        if not isinstance(self.owner_id, Unset):
            owner_id = str(self.owner_id)

        phase: str | Unset = UNSET
        if not isinstance(self.phase, Unset):
            phase = self.phase.value

        pool_name = self.pool_name

        ports: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.ports, Unset):
            ports = []
            for ports_item_data in self.ports:
                ports_item = ports_item_data.to_dict()
                ports.append(ports_item)

        ready_replicas = self.ready_replicas

        replicas = self.replicas

        resources: dict[str, Any] | Unset = UNSET
        if not isinstance(self.resources, Unset):
            resources = self.resources.to_dict()

        route: dict[str, Any] | Unset = UNSET
        if not isinstance(self.route, Unset):
            route = self.route.to_dict()

        tenant_display_name = self.tenant_display_name

        unit_name = self.unit_name

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "id": id,
                "name": name,
                "namespace": namespace,
                "owner": owner,
                "tenantName": tenant_name,
                "updatedAt": updated_at,
            }
        )
        if access_url is not UNSET:
            field_dict["accessUrl"] = access_url
        if args is not UNSET:
            field_dict["args"] = args
        if backend is not UNSET:
            field_dict["backend"] = backend
        if command is not UNSET:
            field_dict["command"] = command
        if compute_namespace is not UNSET:
            field_dict["computeNamespace"] = compute_namespace
        if description is not UNSET:
            field_dict["description"] = description
        if desired_state is not UNSET:
            field_dict["desiredState"] = desired_state
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if env is not UNSET:
            field_dict["env"] = env
        if image is not UNSET:
            field_dict["image"] = image
        if message is not UNSET:
            field_dict["message"] = message
        if model_name is not UNSET:
            field_dict["modelName"] = model_name
        if model_version is not UNSET:
            field_dict["modelVersion"] = model_version
        if owner_id is not UNSET:
            field_dict["ownerId"] = owner_id
        if phase is not UNSET:
            field_dict["phase"] = phase
        if pool_name is not UNSET:
            field_dict["poolName"] = pool_name
        if ports is not UNSET:
            field_dict["ports"] = ports
        if ready_replicas is not UNSET:
            field_dict["readyReplicas"] = ready_replicas
        if replicas is not UNSET:
            field_dict["replicas"] = replicas
        if resources is not UNSET:
            field_dict["resources"] = resources
        if route is not UNSET:
            field_dict["route"] = route
        if tenant_display_name is not UNSET:
            field_dict["tenantDisplayName"] = tenant_display_name
        if unit_name is not UNSET:
            field_dict["unitName"] = unit_name

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.backend import Backend
        from ..models.env_var import EnvVar
        from ..models.ml_service_route import MLServiceRoute
        from ..models.resource_map import ResourceMap
        from ..models.service_port import ServicePort

        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        id = UUID(d.pop("id"))

        name = d.pop("name")

        namespace = d.pop("namespace")

        owner = d.pop("owner")

        tenant_name = d.pop("tenantName")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

        access_url = d.pop("accessUrl", UNSET)

        args = cast(list[str], d.pop("args", UNSET))

        _backend = d.pop("backend", UNSET)
        backend: Backend | Unset
        if isinstance(_backend, Unset):
            backend = UNSET
        else:
            backend = Backend.from_dict(_backend)

        command = cast(list[str], d.pop("command", UNSET))

        compute_namespace = d.pop("computeNamespace", UNSET)

        description = d.pop("description", UNSET)

        _desired_state = d.pop("desiredState", UNSET)
        desired_state: MLServiceDesiredState | Unset
        if isinstance(_desired_state, Unset):
            desired_state = UNSET
        else:
            desired_state = MLServiceDesiredState(_desired_state)

        display_name = d.pop("displayName", UNSET)

        _env = d.pop("env", UNSET)
        env: list[EnvVar] | Unset = UNSET
        if _env is not UNSET:
            env = []
            for env_item_data in _env:
                env_item = EnvVar.from_dict(env_item_data)

                env.append(env_item)

        image = d.pop("image", UNSET)

        message = d.pop("message", UNSET)

        model_name = d.pop("modelName", UNSET)

        model_version = d.pop("modelVersion", UNSET)

        _owner_id = d.pop("ownerId", UNSET)
        owner_id: UUID | Unset
        if isinstance(_owner_id, Unset):
            owner_id = UNSET
        else:
            owner_id = UUID(_owner_id)

        _phase = d.pop("phase", UNSET)
        phase: MLServicePhase | Unset
        if isinstance(_phase, Unset):
            phase = UNSET
        else:
            phase = MLServicePhase(_phase)

        pool_name = d.pop("poolName", UNSET)

        _ports = d.pop("ports", UNSET)
        ports: list[ServicePort] | Unset = UNSET
        if _ports is not UNSET:
            ports = []
            for ports_item_data in _ports:
                ports_item = ServicePort.from_dict(ports_item_data)

                ports.append(ports_item)

        ready_replicas = d.pop("readyReplicas", UNSET)

        replicas = d.pop("replicas", UNSET)

        _resources = d.pop("resources", UNSET)
        resources: ResourceMap | Unset
        if isinstance(_resources, Unset):
            resources = UNSET
        else:
            resources = ResourceMap.from_dict(_resources)

        _route = d.pop("route", UNSET)
        route: MLServiceRoute | Unset
        if isinstance(_route, Unset):
            route = UNSET
        else:
            route = MLServiceRoute.from_dict(_route)

        tenant_display_name = d.pop("tenantDisplayName", UNSET)

        unit_name = d.pop("unitName", UNSET)

        ml_service = cls(
            created_at=created_at,
            id=id,
            name=name,
            namespace=namespace,
            owner=owner,
            tenant_name=tenant_name,
            updated_at=updated_at,
            access_url=access_url,
            args=args,
            backend=backend,
            command=command,
            compute_namespace=compute_namespace,
            description=description,
            desired_state=desired_state,
            display_name=display_name,
            env=env,
            image=image,
            message=message,
            model_name=model_name,
            model_version=model_version,
            owner_id=owner_id,
            phase=phase,
            pool_name=pool_name,
            ports=ports,
            ready_replicas=ready_replicas,
            replicas=replicas,
            resources=resources,
            route=route,
            tenant_display_name=tenant_display_name,
            unit_name=unit_name,
        )

        ml_service.additional_properties = d
        return ml_service

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
