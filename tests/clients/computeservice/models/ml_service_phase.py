from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="MLServicePhase")


@_attrs_define
class MLServicePhase:
    """
    Attributes:
        generation (int): Desired-state generation, bumped on every spec-affecting change (scale).
        name (str): MLService name, unique within the namespace.
        observed_generation (int): Generation the operator last reconciled; equals generation when in sync.
        phase (str): Current service lifecycle phase: Creating, Pending, Ready, Degraded, Failed, Deleting, Deleted.
        ready_replicas (int): Number of replicas that have passed readiness.
        message (str | Unset): Human-readable status detail for the current phase.
    """

    generation: int
    name: str
    observed_generation: int
    phase: str
    ready_replicas: int
    message: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        generation = self.generation

        name = self.name

        observed_generation = self.observed_generation

        phase = self.phase

        ready_replicas = self.ready_replicas

        message = self.message

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "generation": generation,
                "name": name,
                "observedGeneration": observed_generation,
                "phase": phase,
                "readyReplicas": ready_replicas,
            }
        )
        if message is not UNSET:
            field_dict["message"] = message

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        generation = d.pop("generation")

        name = d.pop("name")

        observed_generation = d.pop("observedGeneration")

        phase = d.pop("phase")

        ready_replicas = d.pop("readyReplicas")

        message = d.pop("message", UNSET)

        ml_service_phase = cls(
            generation=generation,
            name=name,
            observed_generation=observed_generation,
            phase=phase,
            ready_replicas=ready_replicas,
            message=message,
        )

        ml_service_phase.additional_properties = d
        return ml_service_phase

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
