from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.run_phase import RunPhase
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.backend import Backend
    from ..models.ml_run_role_status import MLRunRoleStatus
    from ..models.ml_run_spec import MLRunSpec
    from ..models.resource_map import ResourceMap
    from ..models.run_policy import RunPolicy


T = TypeVar("T", bound="Run")


@_attrs_define
class Run:
    """
    Example:
        {'backend': {'engine': 'pytorchjob', 'name': 'native'}, 'computeNamespace': 'axisml-team-vision', 'createdAt':
            '2026-06-28T09:00:00Z', 'description': 'Distributed ResNet-50 training run on ImageNet.', 'displayName':
            'ResNet-50 Training #7', 'id': 'b7d9e3f1-1a2b-3c4d-5e6f-708192a3b4c5', 'jobName': 'resnet-train', 'message':
            'All worker replicas ready.', 'name': 'resnet-train-7', 'namespace': 'team-vision', 'owner': 'li.wei',
            'ownerId': '3a2b1c0d-4e5f-6789-abcd-ef0123456789', 'phase': 'Running', 'poolName': 'gpu-a100', 'resources':
            {'cpu': '32', 'memory': '256Gi', 'nvidia.com/gpu': '8'}, 'roles': [{'activeReplicas': 4, 'failedReplicas': 0,
            'name': 'worker', 'readyReplicas': 4, 'replicas': 4, 'restartPolicy': 'OnFailure', 'succeededReplicas': 0,
            'template': {'args': ['--epochs', '90', '--batch-size', '256'], 'command': ['python', 'train.py'], 'env':
            [{'name': 'NCCL_DEBUG', 'value': 'INFO'}], 'image': 'registry.axisml.io/training/resnet:1.4.0', 'ports':
            [{'containerPort': 8080, 'name': 'http', 'protocol': 'TCP'}], 'resources': {'cpu': '8', 'memory': '64Gi',
            'nvidia.com/gpu': '2'}, 'volumeMounts': [{'mountPath': '/data', 'name': 'data'}], 'volumes': [{'name': 'data',
            'persistentVolumeClaim': {'claimName': 'resnet-imagenet'}}]}}], 'runNumber': 7, 'runPolicy':
            {'activeDeadlineSeconds': 86400, 'backoffLimit': 2, 'progressDeadlineSeconds': 600, 'ttlSecondsAfterFinished':
            3600}, 'scheduledAt': '2026-06-28T09:00:00Z', 'spec': {'backend': {'engine': 'pytorchjob', 'name': 'native'},
            'roles': [{'name': 'worker', 'replicas': 4, 'restartPolicy': 'OnFailure', 'template': {'args': ['--epochs',
            '90', '--batch-size', '256'], 'command': ['python', 'train.py'], 'env': [{'name': 'NCCL_DEBUG', 'value':
            'INFO'}], 'image': 'registry.axisml.io/training/resnet:1.4.0', 'ports': [{'containerPort': 8080, 'name': 'http',
            'protocol': 'TCP'}], 'resources': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}, 'volumeMounts':
            [{'mountPath': '/data', 'name': 'data'}], 'volumes': [{'name': 'data', 'persistentVolumeClaim': {'claimName':
            'resnet-imagenet'}}]}}], 'runPolicy': {'activeDeadlineSeconds': 86400, 'backoffLimit': 2,
            'progressDeadlineSeconds': 600, 'ttlSecondsAfterFinished': 3600}, 'scheduling': {'minMember': 4,
            'priorityClass': 'high-priority', 'quota': 'axisml-team-vision-gpu-a100'}}, 'startedAt': '2026-06-28T09:00:00Z',
            'tenantDisplayName': 'Vision Team', 'tenantName': 'team-vision', 'unitName': 'a100-2x', 'updatedAt':
            '2026-06-28T09:30:00Z'}

    Attributes:
        backend (Backend):  Example: {'engine': 'pytorchjob', 'name': 'native'}.
        created_at (datetime.datetime): Time the run was created.
        name (str): Run name (<job>-<n>).
        namespace (str): Platform tenant namespace the run belongs to.
        owner (str): Username of the run owner.
        tenant_name (str): Tenant identifier owning the run.
        updated_at (datetime.datetime): Time the run was last updated.
        compute_namespace (str | Unset): Underlying compute (Kubernetes) namespace executing the run.
        description (str | Unset): Free-text run description.
        display_name (str | Unset): Human-readable run label.
        finished_at (datetime.datetime | None | Unset): Time the run reached a terminal phase.
        id (UUID | Unset): Stable run identifier.
        job_name (str | Unset): Name of the Job definition that produced this run.
        message (str | Unset): Human-readable status detail for the current phase.
        owner_id (UUID | Unset): User ID of the run owner.
        phase (RunPhase | Unset): Run (compute MLRun) phase. The active (non-terminal) phases — Creating / Pending /
            Running / Canceling — block Job-definition deletion.
        pool_name (str | Unset): Resource pool the run is scheduled onto.
        resources (ResourceMap | Unset): Kubernetes-style resource quantity map (e.g., {"cpu": "100", "memory": "1Ti",
            "nvidia.com/gpu": "8"}).
        roles (list[MLRunRoleStatus] | Unset): Live per-role status.
        run_number (int | Unset): Monotonic per-job run sequence number.
        run_policy (RunPolicy | Unset):  Example: {'activeDeadlineSeconds': 86400, 'backoffLimit': 2,
            'progressDeadlineSeconds': 600, 'ttlSecondsAfterFinished': 3600}.
        scheduled_at (datetime.datetime | None | Unset): Time the run was admitted by the scheduler (left Pending).
        spec (MLRunSpec | Unset):  Example: {'backend': {'engine': 'pytorchjob', 'name': 'native'}, 'roles': [{'name':
            'worker', 'replicas': 4, 'restartPolicy': 'OnFailure', 'template': {'args': ['--epochs', '90', '--batch-size',
            '256'], 'command': ['python', 'train.py'], 'env': [{'name': 'NCCL_DEBUG', 'value': 'INFO'}], 'image':
            'registry.axisml.io/training/resnet:1.4.0', 'ports': [{'containerPort': 8080, 'name': 'http', 'protocol':
            'TCP'}], 'resources': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}, 'volumeMounts': [{'mountPath':
            '/data', 'name': 'data'}], 'volumes': [{'name': 'data', 'persistentVolumeClaim': {'claimName': 'resnet-
            imagenet'}}]}}], 'runPolicy': {'activeDeadlineSeconds': 86400, 'backoffLimit': 2, 'progressDeadlineSeconds':
            600, 'ttlSecondsAfterFinished': 3600}, 'scheduling': {'minMember': 4, 'priorityClass': 'high-priority', 'quota':
            'axisml-team-vision-gpu-a100'}}.
        started_at (datetime.datetime | None | Unset): Time the run started executing.
        tenant_display_name (str | Unset): Human-readable tenant name.
        unit_name (str | Unset): Resource unit (shape) within the pool.
    """

    backend: Backend
    created_at: datetime.datetime
    name: str
    namespace: str
    owner: str
    tenant_name: str
    updated_at: datetime.datetime
    compute_namespace: str | Unset = UNSET
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    finished_at: datetime.datetime | None | Unset = UNSET
    id: UUID | Unset = UNSET
    job_name: str | Unset = UNSET
    message: str | Unset = UNSET
    owner_id: UUID | Unset = UNSET
    phase: RunPhase | Unset = UNSET
    pool_name: str | Unset = UNSET
    resources: ResourceMap | Unset = UNSET
    roles: list[MLRunRoleStatus] | Unset = UNSET
    run_number: int | Unset = UNSET
    run_policy: RunPolicy | Unset = UNSET
    scheduled_at: datetime.datetime | None | Unset = UNSET
    spec: MLRunSpec | Unset = UNSET
    started_at: datetime.datetime | None | Unset = UNSET
    tenant_display_name: str | Unset = UNSET
    unit_name: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        backend = self.backend.to_dict()

        created_at = self.created_at.isoformat()

        name = self.name

        namespace = self.namespace

        owner = self.owner

        tenant_name = self.tenant_name

        updated_at = self.updated_at.isoformat()

        compute_namespace = self.compute_namespace

        description = self.description

        display_name = self.display_name

        finished_at: None | str | Unset
        if isinstance(self.finished_at, Unset):
            finished_at = UNSET
        elif isinstance(self.finished_at, datetime.datetime):
            finished_at = self.finished_at.isoformat()
        else:
            finished_at = self.finished_at

        id: str | Unset = UNSET
        if not isinstance(self.id, Unset):
            id = str(self.id)

        job_name = self.job_name

        message = self.message

        owner_id: str | Unset = UNSET
        if not isinstance(self.owner_id, Unset):
            owner_id = str(self.owner_id)

        phase: str | Unset = UNSET
        if not isinstance(self.phase, Unset):
            phase = self.phase.value

        pool_name = self.pool_name

        resources: dict[str, Any] | Unset = UNSET
        if not isinstance(self.resources, Unset):
            resources = self.resources.to_dict()

        roles: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.roles, Unset):
            roles = []
            for roles_item_data in self.roles:
                roles_item = roles_item_data.to_dict()
                roles.append(roles_item)

        run_number = self.run_number

        run_policy: dict[str, Any] | Unset = UNSET
        if not isinstance(self.run_policy, Unset):
            run_policy = self.run_policy.to_dict()

        scheduled_at: None | str | Unset
        if isinstance(self.scheduled_at, Unset):
            scheduled_at = UNSET
        elif isinstance(self.scheduled_at, datetime.datetime):
            scheduled_at = self.scheduled_at.isoformat()
        else:
            scheduled_at = self.scheduled_at

        spec: dict[str, Any] | Unset = UNSET
        if not isinstance(self.spec, Unset):
            spec = self.spec.to_dict()

        started_at: None | str | Unset
        if isinstance(self.started_at, Unset):
            started_at = UNSET
        elif isinstance(self.started_at, datetime.datetime):
            started_at = self.started_at.isoformat()
        else:
            started_at = self.started_at

        tenant_display_name = self.tenant_display_name

        unit_name = self.unit_name

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "backend": backend,
                "createdAt": created_at,
                "name": name,
                "namespace": namespace,
                "owner": owner,
                "tenantName": tenant_name,
                "updatedAt": updated_at,
            }
        )
        if compute_namespace is not UNSET:
            field_dict["computeNamespace"] = compute_namespace
        if description is not UNSET:
            field_dict["description"] = description
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if finished_at is not UNSET:
            field_dict["finishedAt"] = finished_at
        if id is not UNSET:
            field_dict["id"] = id
        if job_name is not UNSET:
            field_dict["jobName"] = job_name
        if message is not UNSET:
            field_dict["message"] = message
        if owner_id is not UNSET:
            field_dict["ownerId"] = owner_id
        if phase is not UNSET:
            field_dict["phase"] = phase
        if pool_name is not UNSET:
            field_dict["poolName"] = pool_name
        if resources is not UNSET:
            field_dict["resources"] = resources
        if roles is not UNSET:
            field_dict["roles"] = roles
        if run_number is not UNSET:
            field_dict["runNumber"] = run_number
        if run_policy is not UNSET:
            field_dict["runPolicy"] = run_policy
        if scheduled_at is not UNSET:
            field_dict["scheduledAt"] = scheduled_at
        if spec is not UNSET:
            field_dict["spec"] = spec
        if started_at is not UNSET:
            field_dict["startedAt"] = started_at
        if tenant_display_name is not UNSET:
            field_dict["tenantDisplayName"] = tenant_display_name
        if unit_name is not UNSET:
            field_dict["unitName"] = unit_name

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.backend import Backend
        from ..models.ml_run_role_status import MLRunRoleStatus
        from ..models.ml_run_spec import MLRunSpec
        from ..models.resource_map import ResourceMap
        from ..models.run_policy import RunPolicy

        d = dict(src_dict)
        backend = Backend.from_dict(d.pop("backend"))

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        name = d.pop("name")

        namespace = d.pop("namespace")

        owner = d.pop("owner")

        tenant_name = d.pop("tenantName")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

        compute_namespace = d.pop("computeNamespace", UNSET)

        description = d.pop("description", UNSET)

        display_name = d.pop("displayName", UNSET)

        def _parse_finished_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                finished_at_type_0 = datetime.datetime.fromisoformat(data)

                return finished_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        finished_at = _parse_finished_at(d.pop("finishedAt", UNSET))

        _id = d.pop("id", UNSET)
        id: UUID | Unset
        if isinstance(_id, Unset):
            id = UNSET
        else:
            id = UUID(_id)

        job_name = d.pop("jobName", UNSET)

        message = d.pop("message", UNSET)

        _owner_id = d.pop("ownerId", UNSET)
        owner_id: UUID | Unset
        if isinstance(_owner_id, Unset):
            owner_id = UNSET
        else:
            owner_id = UUID(_owner_id)

        _phase = d.pop("phase", UNSET)
        phase: RunPhase | Unset
        if isinstance(_phase, Unset):
            phase = UNSET
        else:
            phase = RunPhase(_phase)

        pool_name = d.pop("poolName", UNSET)

        _resources = d.pop("resources", UNSET)
        resources: ResourceMap | Unset
        if isinstance(_resources, Unset):
            resources = UNSET
        else:
            resources = ResourceMap.from_dict(_resources)

        _roles = d.pop("roles", UNSET)
        roles: list[MLRunRoleStatus] | Unset = UNSET
        if _roles is not UNSET:
            roles = []
            for roles_item_data in _roles:
                roles_item = MLRunRoleStatus.from_dict(roles_item_data)

                roles.append(roles_item)

        run_number = d.pop("runNumber", UNSET)

        _run_policy = d.pop("runPolicy", UNSET)
        run_policy: RunPolicy | Unset
        if isinstance(_run_policy, Unset):
            run_policy = UNSET
        else:
            run_policy = RunPolicy.from_dict(_run_policy)

        def _parse_scheduled_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                scheduled_at_type_0 = datetime.datetime.fromisoformat(data)

                return scheduled_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        scheduled_at = _parse_scheduled_at(d.pop("scheduledAt", UNSET))

        _spec = d.pop("spec", UNSET)
        spec: MLRunSpec | Unset
        if isinstance(_spec, Unset):
            spec = UNSET
        else:
            spec = MLRunSpec.from_dict(_spec)

        def _parse_started_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                started_at_type_0 = datetime.datetime.fromisoformat(data)

                return started_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        started_at = _parse_started_at(d.pop("startedAt", UNSET))

        tenant_display_name = d.pop("tenantDisplayName", UNSET)

        unit_name = d.pop("unitName", UNSET)

        run = cls(
            backend=backend,
            created_at=created_at,
            name=name,
            namespace=namespace,
            owner=owner,
            tenant_name=tenant_name,
            updated_at=updated_at,
            compute_namespace=compute_namespace,
            description=description,
            display_name=display_name,
            finished_at=finished_at,
            id=id,
            job_name=job_name,
            message=message,
            owner_id=owner_id,
            phase=phase,
            pool_name=pool_name,
            resources=resources,
            roles=roles,
            run_number=run_number,
            run_policy=run_policy,
            scheduled_at=scheduled_at,
            spec=spec,
            started_at=started_at,
            tenant_display_name=tenant_display_name,
            unit_name=unit_name,
        )

        run.additional_properties = d
        return run

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
