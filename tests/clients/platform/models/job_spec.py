from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.artifact_ref import ArtifactRef
    from ..models.backend import Backend
    from ..models.ml_run_role import MLRunRole
    from ..models.run_policy import RunPolicy


T = TypeVar("T", bound="JobSpec")


@_attrs_define
class JobSpec:
    """
    Example:
        {'artifacts': [{'kind': 'model', 'name': 'resnet50', 'version': '1.4.0'}], 'backend': {'engine': 'pytorchjob',
            'name': 'native'}, 'poolName': 'gpu-a100', 'roles': [{'name': 'worker', 'replicas': 4, 'restartPolicy':
            'OnFailure', 'template': {'args': ['--epochs', '90', '--batch-size', '256'], 'command': ['python', 'train.py'],
            'env': [{'name': 'NCCL_DEBUG', 'value': 'INFO'}], 'image': 'registry.axisml.io/training/resnet:1.4.0', 'ports':
            [{'containerPort': 8080, 'name': 'http', 'protocol': 'TCP'}], 'resources': {'cpu': '8', 'memory': '64Gi',
            'nvidia.com/gpu': '2'}, 'volumeMounts': [{'mountPath': '/data', 'name': 'data'}], 'volumes': [{'name': 'data',
            'persistentVolumeClaim': {'claimName': 'resnet-imagenet'}}]}}], 'runPolicy': {'activeDeadlineSeconds': 86400,
            'backoffLimit': 2, 'progressDeadlineSeconds': 600, 'ttlSecondsAfterFinished': 3600}, 'unitName': 'a100-2x'}

    Attributes:
        backend (Backend):  Example: {'engine': 'pytorchjob', 'name': 'native'}.
        roles (list[MLRunRole]): Run topology roles (at least one).
        artifacts (list[ArtifactRef] | Unset): Model/image artifact versions the runs consume.
        pool_name (str | Unset): Default resource pool for runs.
        run_policy (RunPolicy | Unset):  Example: {'activeDeadlineSeconds': 86400, 'backoffLimit': 2,
            'progressDeadlineSeconds': 600, 'ttlSecondsAfterFinished': 3600}.
        unit_name (str | Unset): Default resource unit (shape) within the pool.
    """

    backend: Backend
    roles: list[MLRunRole]
    artifacts: list[ArtifactRef] | Unset = UNSET
    pool_name: str | Unset = UNSET
    run_policy: RunPolicy | Unset = UNSET
    unit_name: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        backend = self.backend.to_dict()

        roles = []
        for roles_item_data in self.roles:
            roles_item = roles_item_data.to_dict()
            roles.append(roles_item)

        artifacts: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.artifacts, Unset):
            artifacts = []
            for artifacts_item_data in self.artifacts:
                artifacts_item = artifacts_item_data.to_dict()
                artifacts.append(artifacts_item)

        pool_name = self.pool_name

        run_policy: dict[str, Any] | Unset = UNSET
        if not isinstance(self.run_policy, Unset):
            run_policy = self.run_policy.to_dict()

        unit_name = self.unit_name

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "backend": backend,
                "roles": roles,
            }
        )
        if artifacts is not UNSET:
            field_dict["artifacts"] = artifacts
        if pool_name is not UNSET:
            field_dict["poolName"] = pool_name
        if run_policy is not UNSET:
            field_dict["runPolicy"] = run_policy
        if unit_name is not UNSET:
            field_dict["unitName"] = unit_name

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.artifact_ref import ArtifactRef
        from ..models.backend import Backend
        from ..models.ml_run_role import MLRunRole
        from ..models.run_policy import RunPolicy

        d = dict(src_dict)
        backend = Backend.from_dict(d.pop("backend"))

        roles = []
        _roles = d.pop("roles")
        for roles_item_data in _roles:
            roles_item = MLRunRole.from_dict(roles_item_data)

            roles.append(roles_item)

        _artifacts = d.pop("artifacts", UNSET)
        artifacts: list[ArtifactRef] | Unset = UNSET
        if _artifacts is not UNSET:
            artifacts = []
            for artifacts_item_data in _artifacts:
                artifacts_item = ArtifactRef.from_dict(artifacts_item_data)

                artifacts.append(artifacts_item)

        pool_name = d.pop("poolName", UNSET)

        _run_policy = d.pop("runPolicy", UNSET)
        run_policy: RunPolicy | Unset
        if isinstance(_run_policy, Unset):
            run_policy = UNSET
        else:
            run_policy = RunPolicy.from_dict(_run_policy)

        unit_name = d.pop("unitName", UNSET)

        job_spec = cls(
            backend=backend,
            roles=roles,
            artifacts=artifacts,
            pool_name=pool_name,
            run_policy=run_policy,
            unit_name=unit_name,
        )

        job_spec.additional_properties = d
        return job_spec

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
