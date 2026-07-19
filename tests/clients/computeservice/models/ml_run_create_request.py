from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ml_run_backend_spec import MLRunBackendSpec
    from ..models.ml_run_create_request_annotations import MLRunCreateRequestAnnotations
    from ..models.ml_run_create_request_labels import MLRunCreateRequestLabels
    from ..models.ml_run_role_spec import MLRunRoleSpec
    from ..models.ml_run_run_policy_spec import MLRunRunPolicySpec
    from ..models.workloadconfig_config_map import WorkloadconfigConfigMap


T = TypeVar("T", bound="MLRunCreateRequest")


@_attrs_define
class MLRunCreateRequest:
    r"""
    Example:
        {'backend': {'engine': 'pytorchjob', 'name': 'kubeflow-trainer'}, 'configMaps': [{'data': {'trainer.yaml':
            'epochs: 90\nbatchSize: 256\n'}, 'name': 'trainer-config'}], 'description': 'Distributed ResNet-50 training on
            ImageNet.', 'displayName': 'ResNet-50 Training #7', 'labels': {'team': 'vision'}, 'name': 'resnet-train-7',
            'poolName': 'gpu-a100', 'roles': [{'name': 'worker', 'replicas': 4, 'restartPolicy': 'OnFailure', 'template':
            {'args': ['--epochs', '90', '--batch-size', '256'], 'command': ['python', 'train.py'], 'env': [{'name':
            'NCCL_DEBUG', 'value': 'INFO'}], 'image': 'registry.axisml.io/training/resnet:1.4.0', 'resources': {'limits':
            {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}, 'requests': {'cpu': '8', 'memory': '64Gi',
            'nvidia.com/gpu': '2'}}}}], 'runPolicy': {'activeDeadlineSeconds': 86400, 'backoffLimit': 2,
            'ttlSecondsAfterFinished': 3600}, 'unitName': 'a100-2x'}

    Attributes:
        name (str): MLRun name, unique within the namespace.
        pool_name (str): Resource pool name resolved against the ResourcePool CRD via the Informer cache.
        roles (list[MLRunRoleSpec]): Run topology roles (at least one).
        unit_name (str): Resource unit (shape) name within the selected pool.
        annotations (MLRunCreateRequestAnnotations | Unset): User-defined annotations stored on the row and stamped onto
            the CR.
        backend (MLRunBackendSpec | None | Unset): Compute backend/engine that runs the workload; defaults to (native,
            job) when omitted.
        config_maps (list[WorkloadconfigConfigMap] | Unset): ConfigMaps created and owned by this MLRun before its pods
            are reconciled.
        description (str | Unset): Free-text run description.
        display_name (str | Unset): Human-readable run label.
        labels (MLRunCreateRequestLabels | Unset): User-defined labels stored on the row and stamped onto the CR.
        priority_class (str | Unset): Optional Kubernetes PriorityClass name for the run's pods.
        run_policy (MLRunRunPolicySpec | None | Unset): Run-level execution limits (deadline, TTL, backoff).
    """

    name: str
    pool_name: str
    roles: list[MLRunRoleSpec]
    unit_name: str
    annotations: MLRunCreateRequestAnnotations | Unset = UNSET
    backend: MLRunBackendSpec | None | Unset = UNSET
    config_maps: list[WorkloadconfigConfigMap] | Unset = UNSET
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    labels: MLRunCreateRequestLabels | Unset = UNSET
    priority_class: str | Unset = UNSET
    run_policy: MLRunRunPolicySpec | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.ml_run_backend_spec import MLRunBackendSpec
        from ..models.ml_run_run_policy_spec import MLRunRunPolicySpec

        name = self.name

        pool_name = self.pool_name

        roles = []
        for roles_item_data in self.roles:
            roles_item = roles_item_data.to_dict()
            roles.append(roles_item)

        unit_name = self.unit_name

        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        backend: dict[str, Any] | None | Unset
        if isinstance(self.backend, Unset):
            backend = UNSET
        elif isinstance(self.backend, MLRunBackendSpec):
            backend = self.backend.to_dict()
        else:
            backend = self.backend

        config_maps: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.config_maps, Unset):
            config_maps = []
            for config_maps_item_data in self.config_maps:
                config_maps_item = config_maps_item_data.to_dict()
                config_maps.append(config_maps_item)

        description = self.description

        display_name = self.display_name

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        priority_class = self.priority_class

        run_policy: dict[str, Any] | None | Unset
        if isinstance(self.run_policy, Unset):
            run_policy = UNSET
        elif isinstance(self.run_policy, MLRunRunPolicySpec):
            run_policy = self.run_policy.to_dict()
        else:
            run_policy = self.run_policy

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
                "poolName": pool_name,
                "roles": roles,
                "unitName": unit_name,
            }
        )
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if backend is not UNSET:
            field_dict["backend"] = backend
        if config_maps is not UNSET:
            field_dict["configMaps"] = config_maps
        if description is not UNSET:
            field_dict["description"] = description
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if labels is not UNSET:
            field_dict["labels"] = labels
        if priority_class is not UNSET:
            field_dict["priorityClass"] = priority_class
        if run_policy is not UNSET:
            field_dict["runPolicy"] = run_policy

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ml_run_backend_spec import MLRunBackendSpec
        from ..models.ml_run_create_request_annotations import (
            MLRunCreateRequestAnnotations,
        )
        from ..models.ml_run_create_request_labels import MLRunCreateRequestLabels
        from ..models.ml_run_role_spec import MLRunRoleSpec
        from ..models.ml_run_run_policy_spec import MLRunRunPolicySpec
        from ..models.workloadconfig_config_map import WorkloadconfigConfigMap

        d = dict(src_dict)
        name = d.pop("name")

        pool_name = d.pop("poolName")

        roles = []
        _roles = d.pop("roles")
        for roles_item_data in _roles:
            roles_item = MLRunRoleSpec.from_dict(roles_item_data)

            roles.append(roles_item)

        unit_name = d.pop("unitName")

        _annotations = d.pop("annotations", UNSET)
        annotations: MLRunCreateRequestAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = MLRunCreateRequestAnnotations.from_dict(_annotations)

        def _parse_backend(data: object) -> MLRunBackendSpec | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                backend_type_1 = MLRunBackendSpec.from_dict(data)

                return backend_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(MLRunBackendSpec | None | Unset, data)

        backend = _parse_backend(d.pop("backend", UNSET))

        _config_maps = d.pop("configMaps", UNSET)
        config_maps: list[WorkloadconfigConfigMap] | Unset = UNSET
        if _config_maps is not UNSET:
            config_maps = []
            for config_maps_item_data in _config_maps:
                config_maps_item = WorkloadconfigConfigMap.from_dict(
                    config_maps_item_data
                )

                config_maps.append(config_maps_item)

        description = d.pop("description", UNSET)

        display_name = d.pop("displayName", UNSET)

        _labels = d.pop("labels", UNSET)
        labels: MLRunCreateRequestLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = MLRunCreateRequestLabels.from_dict(_labels)

        priority_class = d.pop("priorityClass", UNSET)

        def _parse_run_policy(data: object) -> MLRunRunPolicySpec | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                run_policy_type_1 = MLRunRunPolicySpec.from_dict(data)

                return run_policy_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(MLRunRunPolicySpec | None | Unset, data)

        run_policy = _parse_run_policy(d.pop("runPolicy", UNSET))

        ml_run_create_request = cls(
            name=name,
            pool_name=pool_name,
            roles=roles,
            unit_name=unit_name,
            annotations=annotations,
            backend=backend,
            config_maps=config_maps,
            description=description,
            display_name=display_name,
            labels=labels,
            priority_class=priority_class,
            run_policy=run_policy,
        )

        ml_run_create_request.additional_properties = d
        return ml_run_create_request

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
