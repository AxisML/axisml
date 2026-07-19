from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ml_run_backend_spec import MLRunBackendSpec
    from ..models.ml_run_role_spec import MLRunRoleSpec
    from ..models.ml_run_run_policy_spec import MLRunRunPolicySpec
    from ..models.ml_run_scheduling_spec import MLRunSchedulingSpec
    from ..models.workloadconfig_config_map import WorkloadconfigConfigMap


T = TypeVar("T", bound="MLRunSpec")


@_attrs_define
class MLRunSpec:
    """
    Attributes:
        backend (MLRunBackendSpec):
        roles (list[MLRunRoleSpec]):
        scheduling (MLRunSchedulingSpec):
        config_maps (list[WorkloadconfigConfigMap] | Unset):
        run_policy (MLRunRunPolicySpec | Unset):
    """

    backend: MLRunBackendSpec
    roles: list[MLRunRoleSpec]
    scheduling: MLRunSchedulingSpec
    config_maps: list[WorkloadconfigConfigMap] | Unset = UNSET
    run_policy: MLRunRunPolicySpec | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        backend = self.backend.to_dict()

        roles = []
        for roles_item_data in self.roles:
            roles_item = roles_item_data.to_dict()
            roles.append(roles_item)

        scheduling = self.scheduling.to_dict()

        config_maps: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.config_maps, Unset):
            config_maps = []
            for config_maps_item_data in self.config_maps:
                config_maps_item = config_maps_item_data.to_dict()
                config_maps.append(config_maps_item)

        run_policy: dict[str, Any] | Unset = UNSET
        if not isinstance(self.run_policy, Unset):
            run_policy = self.run_policy.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "backend": backend,
                "roles": roles,
                "scheduling": scheduling,
            }
        )
        if config_maps is not UNSET:
            field_dict["configMaps"] = config_maps
        if run_policy is not UNSET:
            field_dict["runPolicy"] = run_policy

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ml_run_backend_spec import MLRunBackendSpec
        from ..models.ml_run_role_spec import MLRunRoleSpec
        from ..models.ml_run_run_policy_spec import MLRunRunPolicySpec
        from ..models.ml_run_scheduling_spec import MLRunSchedulingSpec
        from ..models.workloadconfig_config_map import WorkloadconfigConfigMap

        d = dict(src_dict)
        backend = MLRunBackendSpec.from_dict(d.pop("backend"))

        roles = []
        _roles = d.pop("roles")
        for roles_item_data in _roles:
            roles_item = MLRunRoleSpec.from_dict(roles_item_data)

            roles.append(roles_item)

        scheduling = MLRunSchedulingSpec.from_dict(d.pop("scheduling"))

        _config_maps = d.pop("configMaps", UNSET)
        config_maps: list[WorkloadconfigConfigMap] | Unset = UNSET
        if _config_maps is not UNSET:
            config_maps = []
            for config_maps_item_data in _config_maps:
                config_maps_item = WorkloadconfigConfigMap.from_dict(
                    config_maps_item_data
                )

                config_maps.append(config_maps_item)

        _run_policy = d.pop("runPolicy", UNSET)
        run_policy: MLRunRunPolicySpec | Unset
        if isinstance(_run_policy, Unset):
            run_policy = UNSET
        else:
            run_policy = MLRunRunPolicySpec.from_dict(_run_policy)

        ml_run_spec = cls(
            backend=backend,
            roles=roles,
            scheduling=scheduling,
            config_maps=config_maps,
            run_policy=run_policy,
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
