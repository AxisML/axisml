from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_toleration import Corev1Toleration
    from ..models.ml_run_scheduling_spec_node_selector import (
        MLRunSchedulingSpecNodeSelector,
    )


T = TypeVar("T", bound="MLRunSchedulingSpec")


@_attrs_define
class MLRunSchedulingSpec:
    """
    Attributes:
        quota (str):
        node_selector (MLRunSchedulingSpecNodeSelector | Unset):
        priority_class (str | Unset):
        tolerations (list[Corev1Toleration] | Unset):
    """

    quota: str
    node_selector: MLRunSchedulingSpecNodeSelector | Unset = UNSET
    priority_class: str | Unset = UNSET
    tolerations: list[Corev1Toleration] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        quota = self.quota

        node_selector: dict[str, Any] | Unset = UNSET
        if not isinstance(self.node_selector, Unset):
            node_selector = self.node_selector.to_dict()

        priority_class = self.priority_class

        tolerations: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.tolerations, Unset):
            tolerations = []
            for tolerations_item_data in self.tolerations:
                tolerations_item = tolerations_item_data.to_dict()
                tolerations.append(tolerations_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "quota": quota,
            }
        )
        if node_selector is not UNSET:
            field_dict["nodeSelector"] = node_selector
        if priority_class is not UNSET:
            field_dict["priorityClass"] = priority_class
        if tolerations is not UNSET:
            field_dict["tolerations"] = tolerations

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_toleration import Corev1Toleration
        from ..models.ml_run_scheduling_spec_node_selector import (
            MLRunSchedulingSpecNodeSelector,
        )

        d = dict(src_dict)
        quota = d.pop("quota")

        _node_selector = d.pop("nodeSelector", UNSET)
        node_selector: MLRunSchedulingSpecNodeSelector | Unset
        if isinstance(_node_selector, Unset):
            node_selector = UNSET
        else:
            node_selector = MLRunSchedulingSpecNodeSelector.from_dict(_node_selector)

        priority_class = d.pop("priorityClass", UNSET)

        _tolerations = d.pop("tolerations", UNSET)
        tolerations: list[Corev1Toleration] | Unset = UNSET
        if _tolerations is not UNSET:
            tolerations = []
            for tolerations_item_data in _tolerations:
                tolerations_item = Corev1Toleration.from_dict(tolerations_item_data)

                tolerations.append(tolerations_item)

        ml_run_scheduling_spec = cls(
            quota=quota,
            node_selector=node_selector,
            priority_class=priority_class,
            tolerations=tolerations,
        )

        ml_run_scheduling_spec.additional_properties = d
        return ml_run_scheduling_spec

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
