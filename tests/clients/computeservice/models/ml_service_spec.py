from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ml_service_backend import MLServiceBackend
    from ..models.ml_service_role_spec import MLServiceRoleSpec
    from ..models.ml_service_route import MLServiceRoute
    from ..models.ml_service_run_policy import MLServiceRunPolicy
    from ..models.ml_service_scheduling import MLServiceScheduling


T = TypeVar("T", bound="MLServiceSpec")


@_attrs_define
class MLServiceSpec:
    """
    Attributes:
        backend (MLServiceBackend):
        roles (list[MLServiceRoleSpec]):
        scheduling (MLServiceScheduling):
        route (MLServiceRoute | None | Unset):
        run_policy (MLServiceRunPolicy | Unset):
    """

    backend: MLServiceBackend
    roles: list[MLServiceRoleSpec]
    scheduling: MLServiceScheduling
    route: MLServiceRoute | None | Unset = UNSET
    run_policy: MLServiceRunPolicy | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.ml_service_route import MLServiceRoute

        backend = self.backend.to_dict()

        roles = []
        for roles_item_data in self.roles:
            roles_item = roles_item_data.to_dict()
            roles.append(roles_item)

        scheduling = self.scheduling.to_dict()

        route: dict[str, Any] | None | Unset
        if isinstance(self.route, Unset):
            route = UNSET
        elif isinstance(self.route, MLServiceRoute):
            route = self.route.to_dict()
        else:
            route = self.route

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
        if route is not UNSET:
            field_dict["route"] = route
        if run_policy is not UNSET:
            field_dict["runPolicy"] = run_policy

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ml_service_backend import MLServiceBackend
        from ..models.ml_service_role_spec import MLServiceRoleSpec
        from ..models.ml_service_route import MLServiceRoute
        from ..models.ml_service_run_policy import MLServiceRunPolicy
        from ..models.ml_service_scheduling import MLServiceScheduling

        d = dict(src_dict)
        backend = MLServiceBackend.from_dict(d.pop("backend"))

        roles = []
        _roles = d.pop("roles")
        for roles_item_data in _roles:
            roles_item = MLServiceRoleSpec.from_dict(roles_item_data)

            roles.append(roles_item)

        scheduling = MLServiceScheduling.from_dict(d.pop("scheduling"))

        def _parse_route(data: object) -> MLServiceRoute | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                route_type_1 = MLServiceRoute.from_dict(data)

                return route_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(MLServiceRoute | None | Unset, data)

        route = _parse_route(d.pop("route", UNSET))

        _run_policy = d.pop("runPolicy", UNSET)
        run_policy: MLServiceRunPolicy | Unset
        if isinstance(_run_policy, Unset):
            run_policy = UNSET
        else:
            run_policy = MLServiceRunPolicy.from_dict(_run_policy)

        ml_service_spec = cls(
            backend=backend,
            roles=roles,
            scheduling=scheduling,
            route=route,
            run_policy=run_policy,
        )

        ml_service_spec.additional_properties = d
        return ml_service_spec

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
