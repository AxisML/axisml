from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.artifact_ref import ArtifactRef
    from ..models.resource_map import ResourceMap
    from ..models.run_trigger_request_roles_item import RunTriggerRequestRolesItem
    from ..models.string_map import StringMap


T = TypeVar("T", bound="RunTriggerRequest")


@_attrs_define
class RunTriggerRequest:
    """
    Example:
        {'displayName': 'ResNet-50 Training #8', 'poolName': 'gpu-a100', 'roles': [{'args': ['--epochs', '120'], 'name':
            'worker'}], 'unitName': 'a100-2x'}

    Attributes:
        annotations (StringMap | Unset):
        artifacts (list[ArtifactRef] | Unset): Override artifact versions for this run.
        display_name (str | Unset): Display name for the triggered run.
        labels (StringMap | Unset):
        pool_name (str | Unset): Override resource pool for this run.
        resources (ResourceMap | Unset): Kubernetes-style resource quantity map (e.g., {"cpu": "100", "memory": "1Ti",
            "nvidia.com/gpu": "8"}).
        roles (list[RunTriggerRequestRolesItem] | Unset): Per-role trigger-time overrides.
        unit_name (str | Unset): Override resource unit (shape) for this run.
    """

    annotations: StringMap | Unset = UNSET
    artifacts: list[ArtifactRef] | Unset = UNSET
    display_name: str | Unset = UNSET
    labels: StringMap | Unset = UNSET
    pool_name: str | Unset = UNSET
    resources: ResourceMap | Unset = UNSET
    roles: list[RunTriggerRequestRolesItem] | Unset = UNSET
    unit_name: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        artifacts: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.artifacts, Unset):
            artifacts = []
            for artifacts_item_data in self.artifacts:
                artifacts_item = artifacts_item_data.to_dict()
                artifacts.append(artifacts_item)

        display_name = self.display_name

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

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

        unit_name = self.unit_name

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if artifacts is not UNSET:
            field_dict["artifacts"] = artifacts
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if labels is not UNSET:
            field_dict["labels"] = labels
        if pool_name is not UNSET:
            field_dict["poolName"] = pool_name
        if resources is not UNSET:
            field_dict["resources"] = resources
        if roles is not UNSET:
            field_dict["roles"] = roles
        if unit_name is not UNSET:
            field_dict["unitName"] = unit_name

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.artifact_ref import ArtifactRef
        from ..models.resource_map import ResourceMap
        from ..models.run_trigger_request_roles_item import RunTriggerRequestRolesItem
        from ..models.string_map import StringMap

        d = dict(src_dict)
        _annotations = d.pop("annotations", UNSET)
        annotations: StringMap | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = StringMap.from_dict(_annotations)

        _artifacts = d.pop("artifacts", UNSET)
        artifacts: list[ArtifactRef] | Unset = UNSET
        if _artifacts is not UNSET:
            artifacts = []
            for artifacts_item_data in _artifacts:
                artifacts_item = ArtifactRef.from_dict(artifacts_item_data)

                artifacts.append(artifacts_item)

        display_name = d.pop("displayName", UNSET)

        _labels = d.pop("labels", UNSET)
        labels: StringMap | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = StringMap.from_dict(_labels)

        pool_name = d.pop("poolName", UNSET)

        _resources = d.pop("resources", UNSET)
        resources: ResourceMap | Unset
        if isinstance(_resources, Unset):
            resources = UNSET
        else:
            resources = ResourceMap.from_dict(_resources)

        _roles = d.pop("roles", UNSET)
        roles: list[RunTriggerRequestRolesItem] | Unset = UNSET
        if _roles is not UNSET:
            roles = []
            for roles_item_data in _roles:
                roles_item = RunTriggerRequestRolesItem.from_dict(roles_item_data)

                roles.append(roles_item)

        unit_name = d.pop("unitName", UNSET)

        run_trigger_request = cls(
            annotations=annotations,
            artifacts=artifacts,
            display_name=display_name,
            labels=labels,
            pool_name=pool_name,
            resources=resources,
            roles=roles,
            unit_name=unit_name,
        )

        run_trigger_request.additional_properties = d
        return run_trigger_request

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
