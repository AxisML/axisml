from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.role_template import RoleTemplate


T = TypeVar("T", bound="MLRunRoleStatus")


@_attrs_define
class MLRunRoleStatus:
    """
    Example:
        {'activeReplicas': 4, 'failedReplicas': 0, 'name': 'worker', 'readyReplicas': 4, 'replicas': 4, 'restartPolicy':
            'OnFailure', 'succeededReplicas': 0, 'template': {'args': ['--epochs', '90', '--batch-size', '256'], 'command':
            ['python', 'train.py'], 'env': [{'name': 'NCCL_DEBUG', 'value': 'INFO'}], 'image':
            'registry.axisml.io/training/resnet:1.4.0', 'ports': [{'containerPort': 8080, 'name': 'http', 'protocol':
            'TCP'}], 'resources': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}, 'volumeMounts': [{'mountPath':
            '/data', 'name': 'data'}], 'volumes': [{'name': 'data', 'persistentVolumeClaim': {'claimName': 'resnet-
            imagenet'}}]}}

    Attributes:
        active_replicas (int | Unset): Pods currently running.
        failed_replicas (int | Unset): Pods that terminated in failure.
        name (str | Unset): Role name.
        ready_replicas (int | Unset): Pods that have passed readiness.
        replicas (int | Unset): Desired replica count for the role.
        restart_policy (str | Unset): Effective restart policy for the role.
        succeeded_replicas (int | Unset): Pods that completed successfully.
        template (RoleTemplate | Unset):  Example: {'args': ['--epochs', '90', '--batch-size', '256'], 'command':
            ['python', 'train.py'], 'env': [{'name': 'NCCL_DEBUG', 'value': 'INFO'}], 'image':
            'registry.axisml.io/training/resnet:1.4.0', 'ports': [{'containerPort': 8080, 'name': 'http', 'protocol':
            'TCP'}], 'resources': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}, 'volumeMounts': [{'mountPath':
            '/data', 'name': 'data'}], 'volumes': [{'name': 'data', 'persistentVolumeClaim': {'claimName': 'resnet-
            imagenet'}}]}.
    """

    active_replicas: int | Unset = UNSET
    failed_replicas: int | Unset = UNSET
    name: str | Unset = UNSET
    ready_replicas: int | Unset = UNSET
    replicas: int | Unset = UNSET
    restart_policy: str | Unset = UNSET
    succeeded_replicas: int | Unset = UNSET
    template: RoleTemplate | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        active_replicas = self.active_replicas

        failed_replicas = self.failed_replicas

        name = self.name

        ready_replicas = self.ready_replicas

        replicas = self.replicas

        restart_policy = self.restart_policy

        succeeded_replicas = self.succeeded_replicas

        template: dict[str, Any] | Unset = UNSET
        if not isinstance(self.template, Unset):
            template = self.template.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if active_replicas is not UNSET:
            field_dict["activeReplicas"] = active_replicas
        if failed_replicas is not UNSET:
            field_dict["failedReplicas"] = failed_replicas
        if name is not UNSET:
            field_dict["name"] = name
        if ready_replicas is not UNSET:
            field_dict["readyReplicas"] = ready_replicas
        if replicas is not UNSET:
            field_dict["replicas"] = replicas
        if restart_policy is not UNSET:
            field_dict["restartPolicy"] = restart_policy
        if succeeded_replicas is not UNSET:
            field_dict["succeededReplicas"] = succeeded_replicas
        if template is not UNSET:
            field_dict["template"] = template

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.role_template import RoleTemplate

        d = dict(src_dict)
        active_replicas = d.pop("activeReplicas", UNSET)

        failed_replicas = d.pop("failedReplicas", UNSET)

        name = d.pop("name", UNSET)

        ready_replicas = d.pop("readyReplicas", UNSET)

        replicas = d.pop("replicas", UNSET)

        restart_policy = d.pop("restartPolicy", UNSET)

        succeeded_replicas = d.pop("succeededReplicas", UNSET)

        _template = d.pop("template", UNSET)
        template: RoleTemplate | Unset
        if isinstance(_template, Unset):
            template = UNSET
        else:
            template = RoleTemplate.from_dict(_template)

        ml_run_role_status = cls(
            active_replicas=active_replicas,
            failed_replicas=failed_replicas,
            name=name,
            ready_replicas=ready_replicas,
            replicas=replicas,
            restart_policy=restart_policy,
            succeeded_replicas=succeeded_replicas,
            template=template,
        )

        ml_run_role_status.additional_properties = d
        return ml_run_role_status

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
