from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.tensor_board_phase import TensorBoardPhase
from ..types import UNSET, Unset

T = TypeVar("T", bound="TensorBoard")


@_attrs_define
class TensorBoard:
    """
    Example:
        {'createdAt': '2026-06-20T08:00:00Z', 'message': 'TensorBoard is serving.', 'name': 'bert-finetune-tb', 'phase':
            'Running', 'url': 'https://tb.axisml.io/team-nlp/bert-finetune'}

    Attributes:
        created_at (datetime.datetime): Time the TensorBoard was created.
        name (str): TensorBoard instance name.
        message (str | Unset): Human-readable status detail for the current phase.
        phase (TensorBoardPhase | Unset):
        url (str | Unset): Endpoint URL for the TensorBoard UI once running.
    """

    created_at: datetime.datetime
    name: str
    message: str | Unset = UNSET
    phase: TensorBoardPhase | Unset = UNSET
    url: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        name = self.name

        message = self.message

        phase: str | Unset = UNSET
        if not isinstance(self.phase, Unset):
            phase = self.phase.value

        url = self.url

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "name": name,
            }
        )
        if message is not UNSET:
            field_dict["message"] = message
        if phase is not UNSET:
            field_dict["phase"] = phase
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        name = d.pop("name")

        message = d.pop("message", UNSET)

        _phase = d.pop("phase", UNSET)
        phase: TensorBoardPhase | Unset
        if isinstance(_phase, Unset):
            phase = UNSET
        else:
            phase = TensorBoardPhase(_phase)

        url = d.pop("url", UNSET)

        tensor_board = cls(
            created_at=created_at,
            name=name,
            message=message,
            phase=phase,
            url=url,
        )

        tensor_board.additional_properties = d
        return tensor_board

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
