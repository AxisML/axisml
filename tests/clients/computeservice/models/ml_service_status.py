from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ml_service_condition import MLServiceCondition


T = TypeVar("T", bound="MLServiceStatus")


@_attrs_define
class MLServiceStatus:
    """
    Attributes:
        ready_replicas (int): Number of replicas that have passed readiness.
        conditions (list[MLServiceCondition] | Unset): Operator-reported status conditions.
        endpoint (str | Unset): Resolved external endpoint URL when a route is enabled.
        message (str | Unset): Human-readable status detail for the current phase.
    """

    ready_replicas: int
    conditions: list[MLServiceCondition] | Unset = UNSET
    endpoint: str | Unset = UNSET
    message: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        ready_replicas = self.ready_replicas

        conditions: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.conditions, Unset):
            conditions = []
            for conditions_item_data in self.conditions:
                conditions_item = conditions_item_data.to_dict()
                conditions.append(conditions_item)

        endpoint = self.endpoint

        message = self.message

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "readyReplicas": ready_replicas,
            }
        )
        if conditions is not UNSET:
            field_dict["conditions"] = conditions
        if endpoint is not UNSET:
            field_dict["endpoint"] = endpoint
        if message is not UNSET:
            field_dict["message"] = message

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ml_service_condition import MLServiceCondition

        d = dict(src_dict)
        ready_replicas = d.pop("readyReplicas")

        _conditions = d.pop("conditions", UNSET)
        conditions: list[MLServiceCondition] | Unset = UNSET
        if _conditions is not UNSET:
            conditions = []
            for conditions_item_data in _conditions:
                conditions_item = MLServiceCondition.from_dict(conditions_item_data)

                conditions.append(conditions_item)

        endpoint = d.pop("endpoint", UNSET)

        message = d.pop("message", UNSET)

        ml_service_status = cls(
            ready_replicas=ready_replicas,
            conditions=conditions,
            endpoint=endpoint,
            message=message,
        )

        ml_service_status.additional_properties = d
        return ml_service_status

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
