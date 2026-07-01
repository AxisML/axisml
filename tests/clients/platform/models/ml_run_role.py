from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.ml_run_role_restart_policy import MLRunRoleRestartPolicy
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.role_template import RoleTemplate


T = TypeVar("T", bound="MLRunRole")


@_attrs_define
class MLRunRole:
    """
    Example:
        {'name': 'worker', 'replicas': 4, 'restartPolicy': 'OnFailure', 'template': {'args': ['--epochs', '90', '--
            batch-size', '256'], 'command': ['python', 'train.py'], 'env': [{'name': 'NCCL_DEBUG', 'value': 'INFO'}],
            'image': 'registry.axisml.io/training/resnet:1.4.0', 'ports': [{'containerPort': 8080, 'name': 'http',
            'protocol': 'TCP'}], 'resources': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}, 'volumeMounts':
            [{'mountPath': '/data', 'name': 'data'}], 'volumes': [{'name': 'data', 'persistentVolumeClaim': {'claimName':
            'resnet-imagenet'}}]}}

    Attributes:
        name (str): Role name within the run topology (e.g. master, worker).
        template (RoleTemplate):  Example: {'args': ['--epochs', '90', '--batch-size', '256'], 'command': ['python',
            'train.py'], 'env': [{'name': 'NCCL_DEBUG', 'value': 'INFO'}], 'image':
            'registry.axisml.io/training/resnet:1.4.0', 'ports': [{'containerPort': 8080, 'name': 'http', 'protocol':
            'TCP'}], 'resources': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}, 'volumeMounts': [{'mountPath':
            '/data', 'name': 'data'}], 'volumes': [{'name': 'data', 'persistentVolumeClaim': {'claimName': 'resnet-
            imagenet'}}]}.
        replicas (int | Unset): Number of pods for this role.
        restart_policy (MLRunRoleRestartPolicy | Unset): Pod restart policy for the role.
    """

    name: str
    template: RoleTemplate
    replicas: int | Unset = UNSET
    restart_policy: MLRunRoleRestartPolicy | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        template = self.template.to_dict()

        replicas = self.replicas

        restart_policy: str | Unset = UNSET
        if not isinstance(self.restart_policy, Unset):
            restart_policy = self.restart_policy.value

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
                "template": template,
            }
        )
        if replicas is not UNSET:
            field_dict["replicas"] = replicas
        if restart_policy is not UNSET:
            field_dict["restartPolicy"] = restart_policy

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.role_template import RoleTemplate

        d = dict(src_dict)
        name = d.pop("name")

        template = RoleTemplate.from_dict(d.pop("template"))

        replicas = d.pop("replicas", UNSET)

        _restart_policy = d.pop("restartPolicy", UNSET)
        restart_policy: MLRunRoleRestartPolicy | Unset
        if isinstance(_restart_policy, Unset):
            restart_policy = UNSET
        else:
            restart_policy = MLRunRoleRestartPolicy(_restart_policy)

        ml_run_role = cls(
            name=name,
            template=template,
            replicas=replicas,
            restart_policy=restart_policy,
        )

        ml_run_role.additional_properties = d
        return ml_run_role

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
