from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.metav_1_label_selector_match_labels import (
        Metav1LabelSelectorMatchLabels,
    )
    from ..models.metav_1_label_selector_requirement import (
        Metav1LabelSelectorRequirement,
    )


T = TypeVar("T", bound="Metav1LabelSelector")


@_attrs_define
class Metav1LabelSelector:
    """
    Attributes:
        match_expressions (list[Metav1LabelSelectorRequirement] | Unset):
        match_labels (Metav1LabelSelectorMatchLabels | Unset):
    """

    match_expressions: list[Metav1LabelSelectorRequirement] | Unset = UNSET
    match_labels: Metav1LabelSelectorMatchLabels | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        match_expressions: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.match_expressions, Unset):
            match_expressions = []
            for match_expressions_item_data in self.match_expressions:
                match_expressions_item = match_expressions_item_data.to_dict()
                match_expressions.append(match_expressions_item)

        match_labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.match_labels, Unset):
            match_labels = self.match_labels.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if match_expressions is not UNSET:
            field_dict["matchExpressions"] = match_expressions
        if match_labels is not UNSET:
            field_dict["matchLabels"] = match_labels

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.metav_1_label_selector_match_labels import (
            Metav1LabelSelectorMatchLabels,
        )
        from ..models.metav_1_label_selector_requirement import (
            Metav1LabelSelectorRequirement,
        )

        d = dict(src_dict)
        _match_expressions = d.pop("matchExpressions", UNSET)
        match_expressions: list[Metav1LabelSelectorRequirement] | Unset = UNSET
        if _match_expressions is not UNSET:
            match_expressions = []
            for match_expressions_item_data in _match_expressions:
                match_expressions_item = Metav1LabelSelectorRequirement.from_dict(
                    match_expressions_item_data
                )

                match_expressions.append(match_expressions_item)

        _match_labels = d.pop("matchLabels", UNSET)
        match_labels: Metav1LabelSelectorMatchLabels | Unset
        if isinstance(_match_labels, Unset):
            match_labels = UNSET
        else:
            match_labels = Metav1LabelSelectorMatchLabels.from_dict(_match_labels)

        metav_1_label_selector = cls(
            match_expressions=match_expressions,
            match_labels=match_labels,
        )

        metav_1_label_selector.additional_properties = d
        return metav_1_label_selector

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
