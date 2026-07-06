from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ml_traffic_policy_spec import MLTrafficPolicySpec
    from ..models.traffic_policy_annotations import TrafficPolicyAnnotations
    from ..models.traffic_policy_labels import TrafficPolicyLabels
    from ..models.traffic_policy_status import TrafficPolicyStatus


T = TypeVar("T", bound="TrafficPolicy")


@_attrs_define
class TrafficPolicy:
    """
    Example:
        {'annotations': {'axisml.io/created-by': 'li.wei', 'git-commit': '8c1f4e2'}, 'createdAt':
            '2026-06-28T09:25:00Z', 'description': 'Canary 10% traffic to v2.', 'displayName': 'Llama-3 canary release',
            'generation': 2, 'id': 'd3f2b1c0-9e8d-7c6b-5a4f-3e2d1c0b9a8f', 'labels': {'team': 'vision'}, 'mode': 'canary',
            'name': 'llama3-canary', 'namespace': 'team-vision', 'observedGeneration': 2, 'owner': 'li.wei', 'phase':
            'Ready', 'spec': {'backend': {'engine': 'inference', 'name': 'kserve'}, 'backends': [{'role': 'stable',
            'serviceName': 'llama3-8b', 'weight': 90}, {'role': 'canary', 'serviceName': 'llama3-8b-v2', 'weight': 10}],
            'endpoint': {'auth': {'jwt': {'audience': 'axisml-inference', 'issuer': 'https://auth.axisml.io', 'jwksUri':
            'https://auth.axisml.io/.well-known/jwks.json'}, 'type': 'jwt'}, 'hostname': 'llama3-8b.team-vision.axisml.io',
            'path': '/v1'}, 'mode': 'canary'}, 'status': {'backends': [{'ready': True, 'serviceName': 'llama3-8b', 'weight':
            90}, {'ready': True, 'serviceName': 'llama3-8b-v2', 'weight': 10}], 'endpoint': 'https://llama3-8b.team-
            vision.axisml.io/v1', 'message': 'Route programmed; weights applied.'}, 'updatedAt': '2026-06-28T09:45:00Z'}

    Attributes:
        created_at (datetime.datetime): Time the policy was created.
        generation (int): Desired-state generation, bumped on every spec-affecting change (split, promote, rollback).
        id (UUID): Stable policy identifier (PG row UUID).
        mode (str): Traffic split mode (weighted, canary, bluegreen).
        name (str): Traffic policy name, unique within the namespace.
        namespace (str): Namespace (= tenant identifier) the policy belongs to.
        observed_generation (int): Generation the operator last reconciled; equals generation when in sync.
        phase (str): Current policy lifecycle phase (Pending, Ready, Degraded, Failed).
        spec (MLTrafficPolicySpec):
        status (TrafficPolicyStatus):
        updated_at (datetime.datetime): Time the policy was last updated.
        annotations (TrafficPolicyAnnotations | Unset): User-defined annotations.
        deleted_at (datetime.datetime | None | Unset): Soft-deletion timestamp, set once the policy is deleted.
        description (str | Unset): Free-text policy description.
        display_name (str | Unset): Human-readable policy label.
        labels (TrafficPolicyLabels | Unset): User-defined labels.
        owner (str | Unset): Username of the policy owner.
    """

    created_at: datetime.datetime
    generation: int
    id: UUID
    mode: str
    name: str
    namespace: str
    observed_generation: int
    phase: str
    spec: MLTrafficPolicySpec
    status: TrafficPolicyStatus
    updated_at: datetime.datetime
    annotations: TrafficPolicyAnnotations | Unset = UNSET
    deleted_at: datetime.datetime | None | Unset = UNSET
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    labels: TrafficPolicyLabels | Unset = UNSET
    owner: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        generation = self.generation

        id = str(self.id)

        mode = self.mode

        name = self.name

        namespace = self.namespace

        observed_generation = self.observed_generation

        phase = self.phase

        spec = self.spec.to_dict()

        status = self.status.to_dict()

        updated_at = self.updated_at.isoformat()

        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        deleted_at: None | str | Unset
        if isinstance(self.deleted_at, Unset):
            deleted_at = UNSET
        elif isinstance(self.deleted_at, datetime.datetime):
            deleted_at = self.deleted_at.isoformat()
        else:
            deleted_at = self.deleted_at

        description = self.description

        display_name = self.display_name

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        owner = self.owner

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "generation": generation,
                "id": id,
                "mode": mode,
                "name": name,
                "namespace": namespace,
                "observedGeneration": observed_generation,
                "phase": phase,
                "spec": spec,
                "status": status,
                "updatedAt": updated_at,
            }
        )
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if deleted_at is not UNSET:
            field_dict["deletedAt"] = deleted_at
        if description is not UNSET:
            field_dict["description"] = description
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if labels is not UNSET:
            field_dict["labels"] = labels
        if owner is not UNSET:
            field_dict["owner"] = owner

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ml_traffic_policy_spec import MLTrafficPolicySpec
        from ..models.traffic_policy_annotations import TrafficPolicyAnnotations
        from ..models.traffic_policy_labels import TrafficPolicyLabels
        from ..models.traffic_policy_status import TrafficPolicyStatus

        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        generation = d.pop("generation")

        id = UUID(d.pop("id"))

        mode = d.pop("mode")

        name = d.pop("name")

        namespace = d.pop("namespace")

        observed_generation = d.pop("observedGeneration")

        phase = d.pop("phase")

        spec = MLTrafficPolicySpec.from_dict(d.pop("spec"))

        status = TrafficPolicyStatus.from_dict(d.pop("status"))

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

        _annotations = d.pop("annotations", UNSET)
        annotations: TrafficPolicyAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = TrafficPolicyAnnotations.from_dict(_annotations)

        def _parse_deleted_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                deleted_at_type_0 = datetime.datetime.fromisoformat(data)

                return deleted_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        deleted_at = _parse_deleted_at(d.pop("deletedAt", UNSET))

        description = d.pop("description", UNSET)

        display_name = d.pop("displayName", UNSET)

        _labels = d.pop("labels", UNSET)
        labels: TrafficPolicyLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = TrafficPolicyLabels.from_dict(_labels)

        owner = d.pop("owner", UNSET)

        traffic_policy = cls(
            created_at=created_at,
            generation=generation,
            id=id,
            mode=mode,
            name=name,
            namespace=namespace,
            observed_generation=observed_generation,
            phase=phase,
            spec=spec,
            status=status,
            updated_at=updated_at,
            annotations=annotations,
            deleted_at=deleted_at,
            description=description,
            display_name=display_name,
            labels=labels,
            owner=owner,
        )

        traffic_policy.additional_properties = d
        return traffic_policy

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
