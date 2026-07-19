from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.backend import Backend
    from ..models.ml_run_role import MLRunRole
    from ..models.ml_run_spec_scheduling import MLRunSpecScheduling
    from ..models.run_policy import RunPolicy
    from ..models.workload_config_map import WorkloadConfigMap


T = TypeVar("T", bound="MLRunSpec")


@_attrs_define
class MLRunSpec:
    r"""
    Example:
        {'backend': {'engine': 'pytorchjob', 'name': 'native'}, 'configMaps': [{'data': {'trainer.yaml': 'epochs:
            90\nbatchSize: 256\n'}, 'name': 'resnet-training-config'}], 'roles': [{'name': 'worker', 'replicas': 4,
            'restartPolicy': 'OnFailure', 'template': {'args': ['--epochs', '90', '--batch-size', '256'], 'command':
            ['python', 'train.py'], 'env': [{'name': 'NCCL_DEBUG', 'value': 'INFO'}], 'envFrom': [{'configMapRef': {'name':
            'resnet-training-config'}}], 'image': 'registry.axisml.io/training/resnet:1.4.0', 'ports': [{'containerPort':
            8080, 'name': 'http', 'protocol': 'TCP'}], 'resources': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'},
            'volumeMounts': [{'mountPath': '/data', 'name': 'data'}, {'mountPath': '/etc/axisml', 'name': 'config',
            'readOnly': True}], 'volumes': [{'name': 'data', 'persistentVolumeClaim': {'claimName': 'resnet-imagenet'}},
            {'configMap': {'name': 'resnet-training-config'}, 'name': 'config'}]}}], 'runPolicy': {'activeDeadlineSeconds':
            86400, 'backoffLimit': 2, 'progressDeadlineSeconds': 600, 'ttlSecondsAfterFinished': 3600}, 'scheduling':
            {'minMember': 4, 'priorityClass': 'high-priority', 'quota': 'axisml-team-vision-gpu-a100'}}

    Attributes:
        backend (Backend | Unset):  Example: {'engine': 'pytorchjob', 'name': 'native'}.
        config_maps (list[WorkloadConfigMap] | Unset): ConfigMaps created and owned by the triggered MLRun.
        roles (list[MLRunRole] | Unset): Run topology roles.
        run_policy (RunPolicy | Unset):  Example: {'activeDeadlineSeconds': 86400, 'backoffLimit': 2,
            'progressDeadlineSeconds': 600, 'ttlSecondsAfterFinished': 3600}.
        scheduling (MLRunSpecScheduling | Unset): Scheduling directives (gang, priority, tolerations).
    """

    backend: Backend | Unset = UNSET
    config_maps: list[WorkloadConfigMap] | Unset = UNSET
    roles: list[MLRunRole] | Unset = UNSET
    run_policy: RunPolicy | Unset = UNSET
    scheduling: MLRunSpecScheduling | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        backend: dict[str, Any] | Unset = UNSET
        if not isinstance(self.backend, Unset):
            backend = self.backend.to_dict()

        config_maps: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.config_maps, Unset):
            config_maps = []
            for config_maps_item_data in self.config_maps:
                config_maps_item = config_maps_item_data.to_dict()
                config_maps.append(config_maps_item)

        roles: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.roles, Unset):
            roles = []
            for roles_item_data in self.roles:
                roles_item = roles_item_data.to_dict()
                roles.append(roles_item)

        run_policy: dict[str, Any] | Unset = UNSET
        if not isinstance(self.run_policy, Unset):
            run_policy = self.run_policy.to_dict()

        scheduling: dict[str, Any] | Unset = UNSET
        if not isinstance(self.scheduling, Unset):
            scheduling = self.scheduling.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if backend is not UNSET:
            field_dict["backend"] = backend
        if config_maps is not UNSET:
            field_dict["configMaps"] = config_maps
        if roles is not UNSET:
            field_dict["roles"] = roles
        if run_policy is not UNSET:
            field_dict["runPolicy"] = run_policy
        if scheduling is not UNSET:
            field_dict["scheduling"] = scheduling

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.backend import Backend
        from ..models.ml_run_role import MLRunRole
        from ..models.ml_run_spec_scheduling import MLRunSpecScheduling
        from ..models.run_policy import RunPolicy
        from ..models.workload_config_map import WorkloadConfigMap

        d = dict(src_dict)
        _backend = d.pop("backend", UNSET)
        backend: Backend | Unset
        if isinstance(_backend, Unset):
            backend = UNSET
        else:
            backend = Backend.from_dict(_backend)

        _config_maps = d.pop("configMaps", UNSET)
        config_maps: list[WorkloadConfigMap] | Unset = UNSET
        if _config_maps is not UNSET:
            config_maps = []
            for config_maps_item_data in _config_maps:
                config_maps_item = WorkloadConfigMap.from_dict(config_maps_item_data)

                config_maps.append(config_maps_item)

        _roles = d.pop("roles", UNSET)
        roles: list[MLRunRole] | Unset = UNSET
        if _roles is not UNSET:
            roles = []
            for roles_item_data in _roles:
                roles_item = MLRunRole.from_dict(roles_item_data)

                roles.append(roles_item)

        _run_policy = d.pop("runPolicy", UNSET)
        run_policy: RunPolicy | Unset
        if isinstance(_run_policy, Unset):
            run_policy = UNSET
        else:
            run_policy = RunPolicy.from_dict(_run_policy)

        _scheduling = d.pop("scheduling", UNSET)
        scheduling: MLRunSpecScheduling | Unset
        if isinstance(_scheduling, Unset):
            scheduling = UNSET
        else:
            scheduling = MLRunSpecScheduling.from_dict(_scheduling)

        ml_run_spec = cls(
            backend=backend,
            config_maps=config_maps,
            roles=roles,
            run_policy=run_policy,
            scheduling=scheduling,
        )

        ml_run_spec.additional_properties = d
        return ml_run_spec

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
