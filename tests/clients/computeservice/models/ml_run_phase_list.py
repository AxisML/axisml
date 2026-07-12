from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ml_run_phase import MLRunPhase


T = TypeVar("T", bound="MLRunPhaseList")


@_attrs_define
class MLRunPhaseList:
    """
    Attributes:
        count (int): Number of items returned in this page (len(items)).
        items (list[MLRunPhase]): The page of items for the current offset.
        total (int): Total number of matching items across all pages.
        continue_token (str | Unset): Kubernetes-style continuation token for the next page; empty/absent on the final
            page.
    """

    count: int
    items: list[MLRunPhase]
    total: int
    continue_token: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        count = self.count

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)

        total = self.total

        continue_token = self.continue_token

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "count": count,
                "items": items,
                "total": total,
            }
        )
        if continue_token is not UNSET:
            field_dict["continueToken"] = continue_token

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ml_run_phase import MLRunPhase

        d = dict(src_dict)
        count = d.pop("count")

        items = []
        _items = d.pop("items")
        for items_item_data in _items:
            items_item = MLRunPhase.from_dict(items_item_data)

            items.append(items_item)

        total = d.pop("total")

        continue_token = d.pop("continueToken", UNSET)

        ml_run_phase_list = cls(
            count=count,
            items=items,
            total=total,
            continue_token=continue_token,
        )

        ml_run_phase_list.additional_properties = d
        return ml_run_phase_list

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
