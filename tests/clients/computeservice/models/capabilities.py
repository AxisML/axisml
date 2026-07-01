from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="Capabilities")


@_attrs_define
class Capabilities:
    """
    Example:
        {'quotaEnforcement': True, 'runtime': 'kubernetes'}

    Attributes:
        quota_enforcement (bool): True when the scheduler admits pods against an ElasticQuota (Kubernetes form); false
            on the Lite Standalone runtime.
        runtime (str): Workload execution engine for this deployment form (kubernetes or standalone).
    """

    quota_enforcement: bool
    runtime: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        quota_enforcement = self.quota_enforcement

        runtime = self.runtime

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "quotaEnforcement": quota_enforcement,
                "runtime": runtime,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        quota_enforcement = d.pop("quotaEnforcement")

        runtime = d.pop("runtime")

        capabilities = cls(
            quota_enforcement=quota_enforcement,
            runtime=runtime,
        )

        capabilities.additional_properties = d
        return capabilities

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
