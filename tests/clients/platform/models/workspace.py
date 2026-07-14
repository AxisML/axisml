from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.workspace_desired_state import WorkspaceDesiredState
from ..models.workspace_phase import WorkspacePhase
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.env_var import EnvVar
    from ..models.resource_map import ResourceMap
    from ..models.workspace_endpoint import WorkspaceEndpoint
    from ..models.workspace_lifecycle import WorkspaceLifecycle
    from ..models.workspace_volume import WorkspaceVolume


T = TypeVar("T", bound="Workspace")


@_attrs_define
class Workspace:
    """
    Example:
        {'args': ['--NotebookApp.token='], 'command': ['start-notebook.sh'], 'computeNamespace': 'axisml-team-vision',
            'containerPort': 8888, 'createdAt': '2026-06-20T08:00:00Z', 'description': 'JupyterLab interactive development
            environment.', 'desiredState': 'Running', 'displayName': 'Vision team dev environment', 'endpoint':
            {'accessUrl': 'https://axisml.example.com/ws/team-vision/notebook-dev/', 'internalDns': 'notebook-dev.axisml-
            team-vision.svc.cluster.local', 'tools': [{'label': 'JupyterLab', 'name': 'jupyter', 'url':
            'https://axisml.example.com/ws/team-vision/notebook-dev/lab'}, {'label': 'VS Code', 'name': 'vscode', 'url':
            'https://axisml.example.com/ws/team-vision/notebook-dev/vscode/'}, {'label': 'Terminal', 'name': 'terminal',
            'url': 'https://axisml.example.com/ws/team-vision/notebook-dev/terminals/1'}]}, 'env': [{'name':
            'JUPYTER_ENABLE_LAB', 'value': 'yes'}], 'id': 'f1e2d3c4-5b6a-4798-8c0d-1e2f3a4b5c6d', 'image':
            'registry.axisml.io/dev/jupyter:3.0.0', 'lastStartedAt': '2026-06-28T09:00:00Z', 'lifecycle':
            {'idleTimeoutSeconds': 3600}, 'message': 'Workspace is ready.', 'name': 'notebook-dev', 'namespace': 'team-
            vision', 'owner': 'li.wei', 'ownerId': '3a2b1c0d-4e5f-6789-abcd-ef0123456789', 'phase': 'Running', 'poolName':
            'gpu-a100', 'readyReplicas': 1, 'replicas': 1, 'resources': {'cpu': '4', 'memory': '32Gi', 'nvidia.com/gpu':
            '1'}, 'tenantDisplayName': 'Vision Team', 'tenantName': 'team-vision', 'unitName': 'a100-1x', 'updatedAt':
            '2026-06-28T09:30:00Z', 'volumes': [{'mountPath': '/home/jovyan/work', 'name': 'notebook-data', 'used':
            '12Gi'}]}

    Attributes:
        container_port (int): Port the dev server listens on inside the container.
        created_at (datetime.datetime): Time the workspace was created.
        id (UUID): Stable workspace identifier.
        image (str): Container image for the dev environment (e.g. jupyter, code-server).
        name (str): Workspace name (unique within the tenant).
        namespace (str): Platform tenant namespace the workspace belongs to.
        owner (str): Username of the workspace owner.
        tenant_name (str): Tenant identifier owning the workspace.
        updated_at (datetime.datetime): Time the workspace was last updated.
        args (list[str] | Unset): Container args override.
        command (list[str] | Unset): Container entrypoint override.
        compute_namespace (str | Unset): Underlying compute (Kubernetes) namespace hosting the workspace.
        description (str | Unset): Free-text workspace description.
        desired_state (WorkspaceDesiredState | Unset):
        display_name (str | Unset): Human-readable workspace label.
        endpoint (WorkspaceEndpoint | Unset):  Example: {'accessUrl': 'https://axisml.example.com/ws/team-
            vision/notebook-dev/', 'internalDns': 'notebook-dev.axisml-team-vision.svc.cluster.local', 'tools': [{'label':
            'JupyterLab', 'name': 'jupyter', 'url': 'https://axisml.example.com/ws/team-vision/notebook-dev/lab'}, {'label':
            'VS Code', 'name': 'vscode', 'url': 'https://axisml.example.com/ws/team-vision/notebook-dev/vscode/'}, {'label':
            'Terminal', 'name': 'terminal', 'url': 'https://axisml.example.com/ws/team-vision/notebook-dev/terminals/1'}]}.
        env (list[EnvVar] | Unset): Environment variables injected into the container.
        last_started_at (datetime.datetime | None | Unset): Time the workspace was last started.
        last_stopped_at (datetime.datetime | None | Unset): Time the workspace was last stopped.
        lifecycle (WorkspaceLifecycle | Unset):  Example: {'idleTimeoutSeconds': 3600}.
        message (str | Unset): Human-readable status detail for the current phase.
        owner_id (UUID | Unset): User ID of the workspace owner.
        phase (WorkspacePhase | Unset): Derived from compute service phase + replicas. Hoisted to the top of Workspace
            for B-tree filtering.
        pool_name (str | Unset): Resource pool the workspace is scheduled onto.
        ready_replicas (int | Unset): Pods that have passed readiness.
        replicas (int | Unset): Desired pod count (0 when stopped).
        resources (ResourceMap | Unset): Kubernetes-style resource quantity map (e.g., {"cpu": "100", "memory": "1Ti",
            "nvidia.com/gpu": "8"}).
        tenant_display_name (str | Unset): Human-readable tenant name.
        unit_name (str | Unset): Resource unit (shape) within the pool.
        volumes (list[WorkspaceVolume] | Unset): Data volumes mounted into the workspace.
    """

    container_port: int
    created_at: datetime.datetime
    id: UUID
    image: str
    name: str
    namespace: str
    owner: str
    tenant_name: str
    updated_at: datetime.datetime
    args: list[str] | Unset = UNSET
    command: list[str] | Unset = UNSET
    compute_namespace: str | Unset = UNSET
    description: str | Unset = UNSET
    desired_state: WorkspaceDesiredState | Unset = UNSET
    display_name: str | Unset = UNSET
    endpoint: WorkspaceEndpoint | Unset = UNSET
    env: list[EnvVar] | Unset = UNSET
    last_started_at: datetime.datetime | None | Unset = UNSET
    last_stopped_at: datetime.datetime | None | Unset = UNSET
    lifecycle: WorkspaceLifecycle | Unset = UNSET
    message: str | Unset = UNSET
    owner_id: UUID | Unset = UNSET
    phase: WorkspacePhase | Unset = UNSET
    pool_name: str | Unset = UNSET
    ready_replicas: int | Unset = UNSET
    replicas: int | Unset = UNSET
    resources: ResourceMap | Unset = UNSET
    tenant_display_name: str | Unset = UNSET
    unit_name: str | Unset = UNSET
    volumes: list[WorkspaceVolume] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        container_port = self.container_port

        created_at = self.created_at.isoformat()

        id = str(self.id)

        image = self.image

        name = self.name

        namespace = self.namespace

        owner = self.owner

        tenant_name = self.tenant_name

        updated_at = self.updated_at.isoformat()

        args: list[str] | Unset = UNSET
        if not isinstance(self.args, Unset):
            args = self.args

        command: list[str] | Unset = UNSET
        if not isinstance(self.command, Unset):
            command = self.command

        compute_namespace = self.compute_namespace

        description = self.description

        desired_state: str | Unset = UNSET
        if not isinstance(self.desired_state, Unset):
            desired_state = self.desired_state.value

        display_name = self.display_name

        endpoint: dict[str, Any] | Unset = UNSET
        if not isinstance(self.endpoint, Unset):
            endpoint = self.endpoint.to_dict()

        env: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.env, Unset):
            env = []
            for env_item_data in self.env:
                env_item = env_item_data.to_dict()
                env.append(env_item)

        last_started_at: None | str | Unset
        if isinstance(self.last_started_at, Unset):
            last_started_at = UNSET
        elif isinstance(self.last_started_at, datetime.datetime):
            last_started_at = self.last_started_at.isoformat()
        else:
            last_started_at = self.last_started_at

        last_stopped_at: None | str | Unset
        if isinstance(self.last_stopped_at, Unset):
            last_stopped_at = UNSET
        elif isinstance(self.last_stopped_at, datetime.datetime):
            last_stopped_at = self.last_stopped_at.isoformat()
        else:
            last_stopped_at = self.last_stopped_at

        lifecycle: dict[str, Any] | Unset = UNSET
        if not isinstance(self.lifecycle, Unset):
            lifecycle = self.lifecycle.to_dict()

        message = self.message

        owner_id: str | Unset = UNSET
        if not isinstance(self.owner_id, Unset):
            owner_id = str(self.owner_id)

        phase: str | Unset = UNSET
        if not isinstance(self.phase, Unset):
            phase = self.phase.value

        pool_name = self.pool_name

        ready_replicas = self.ready_replicas

        replicas = self.replicas

        resources: dict[str, Any] | Unset = UNSET
        if not isinstance(self.resources, Unset):
            resources = self.resources.to_dict()

        tenant_display_name = self.tenant_display_name

        unit_name = self.unit_name

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
                "containerPort": container_port,
                "createdAt": created_at,
                "id": id,
                "image": image,
                "name": name,
                "namespace": namespace,
                "owner": owner,
                "tenantName": tenant_name,
                "updatedAt": updated_at,
            }
        )
        if args is not UNSET:
            field_dict["args"] = args
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
        if endpoint is not UNSET:
            field_dict["endpoint"] = endpoint
        if env is not UNSET:
            field_dict["env"] = env
        if last_started_at is not UNSET:
            field_dict["lastStartedAt"] = last_started_at
        if last_stopped_at is not UNSET:
            field_dict["lastStoppedAt"] = last_stopped_at
        if lifecycle is not UNSET:
            field_dict["lifecycle"] = lifecycle
        if message is not UNSET:
            field_dict["message"] = message
        if owner_id is not UNSET:
            field_dict["ownerId"] = owner_id
        if phase is not UNSET:
            field_dict["phase"] = phase
        if pool_name is not UNSET:
            field_dict["poolName"] = pool_name
        if ready_replicas is not UNSET:
            field_dict["readyReplicas"] = ready_replicas
        if replicas is not UNSET:
            field_dict["replicas"] = replicas
        if resources is not UNSET:
            field_dict["resources"] = resources
        if tenant_display_name is not UNSET:
            field_dict["tenantDisplayName"] = tenant_display_name
        if unit_name is not UNSET:
            field_dict["unitName"] = unit_name
        if volumes is not UNSET:
            field_dict["volumes"] = volumes

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.env_var import EnvVar
        from ..models.resource_map import ResourceMap
        from ..models.workspace_endpoint import WorkspaceEndpoint
        from ..models.workspace_lifecycle import WorkspaceLifecycle
        from ..models.workspace_volume import WorkspaceVolume

        d = dict(src_dict)
        container_port = d.pop("containerPort")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        id = UUID(d.pop("id"))

        image = d.pop("image")

        name = d.pop("name")

        namespace = d.pop("namespace")

        owner = d.pop("owner")

        tenant_name = d.pop("tenantName")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

        args = cast(list[str], d.pop("args", UNSET))

        command = cast(list[str], d.pop("command", UNSET))

        compute_namespace = d.pop("computeNamespace", UNSET)

        description = d.pop("description", UNSET)

        _desired_state = d.pop("desiredState", UNSET)
        desired_state: WorkspaceDesiredState | Unset
        if isinstance(_desired_state, Unset):
            desired_state = UNSET
        else:
            desired_state = WorkspaceDesiredState(_desired_state)

        display_name = d.pop("displayName", UNSET)

        _endpoint = d.pop("endpoint", UNSET)
        endpoint: WorkspaceEndpoint | Unset
        if isinstance(_endpoint, Unset):
            endpoint = UNSET
        else:
            endpoint = WorkspaceEndpoint.from_dict(_endpoint)

        _env = d.pop("env", UNSET)
        env: list[EnvVar] | Unset = UNSET
        if _env is not UNSET:
            env = []
            for env_item_data in _env:
                env_item = EnvVar.from_dict(env_item_data)

                env.append(env_item)

        def _parse_last_started_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                last_started_at_type_0 = datetime.datetime.fromisoformat(data)

                return last_started_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        last_started_at = _parse_last_started_at(d.pop("lastStartedAt", UNSET))

        def _parse_last_stopped_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                last_stopped_at_type_0 = datetime.datetime.fromisoformat(data)

                return last_stopped_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        last_stopped_at = _parse_last_stopped_at(d.pop("lastStoppedAt", UNSET))

        _lifecycle = d.pop("lifecycle", UNSET)
        lifecycle: WorkspaceLifecycle | Unset
        if isinstance(_lifecycle, Unset):
            lifecycle = UNSET
        else:
            lifecycle = WorkspaceLifecycle.from_dict(_lifecycle)

        message = d.pop("message", UNSET)

        _owner_id = d.pop("ownerId", UNSET)
        owner_id: UUID | Unset
        if isinstance(_owner_id, Unset):
            owner_id = UNSET
        else:
            owner_id = UUID(_owner_id)

        _phase = d.pop("phase", UNSET)
        phase: WorkspacePhase | Unset
        if isinstance(_phase, Unset):
            phase = UNSET
        else:
            phase = WorkspacePhase(_phase)

        pool_name = d.pop("poolName", UNSET)

        ready_replicas = d.pop("readyReplicas", UNSET)

        replicas = d.pop("replicas", UNSET)

        _resources = d.pop("resources", UNSET)
        resources: ResourceMap | Unset
        if isinstance(_resources, Unset):
            resources = UNSET
        else:
            resources = ResourceMap.from_dict(_resources)

        tenant_display_name = d.pop("tenantDisplayName", UNSET)

        unit_name = d.pop("unitName", UNSET)

        _volumes = d.pop("volumes", UNSET)
        volumes: list[WorkspaceVolume] | Unset = UNSET
        if _volumes is not UNSET:
            volumes = []
            for volumes_item_data in _volumes:
                volumes_item = WorkspaceVolume.from_dict(volumes_item_data)

                volumes.append(volumes_item)

        workspace = cls(
            container_port=container_port,
            created_at=created_at,
            id=id,
            image=image,
            name=name,
            namespace=namespace,
            owner=owner,
            tenant_name=tenant_name,
            updated_at=updated_at,
            args=args,
            command=command,
            compute_namespace=compute_namespace,
            description=description,
            desired_state=desired_state,
            display_name=display_name,
            endpoint=endpoint,
            env=env,
            last_started_at=last_started_at,
            last_stopped_at=last_stopped_at,
            lifecycle=lifecycle,
            message=message,
            owner_id=owner_id,
            phase=phase,
            pool_name=pool_name,
            ready_replicas=ready_replicas,
            replicas=replicas,
            resources=resources,
            tenant_display_name=tenant_display_name,
            unit_name=unit_name,
            volumes=volumes,
        )

        workspace.additional_properties = d
        return workspace

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
