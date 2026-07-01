from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.health_status_status import HealthStatusStatus
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.health_status_components import HealthStatusComponents


T = TypeVar("T", bound="HealthStatus")


@_attrs_define
class HealthStatus:
    """
    Example:
        {'components': {'artifact-hub': 'ok', 'compute-service': 'ok', 'database': 'ok'}, 'status': 'ok'}

    Attributes:
        status (HealthStatusStatus): Overall health state (e.g. ok, degraded).
        components (HealthStatusComponents | Unset): Per-dependency health states keyed by component name.
    """

    status: HealthStatusStatus
    components: HealthStatusComponents | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        status = self.status.value

        components: dict[str, Any] | Unset = UNSET
        if not isinstance(self.components, Unset):
            components = self.components.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "status": status,
            }
        )
        if components is not UNSET:
            field_dict["components"] = components

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.health_status_components import HealthStatusComponents

        d = dict(src_dict)
        status = HealthStatusStatus(d.pop("status"))

        _components = d.pop("components", UNSET)
        components: HealthStatusComponents | Unset
        if isinstance(_components, Unset):
            components = UNSET
        else:
            components = HealthStatusComponents.from_dict(_components)

        health_status = cls(
            status=status,
            components=components,
        )

        health_status.additional_properties = d
        return health_status

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
