from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar
from uuid import UUID

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.traffic_policy_mode import TrafficPolicyMode
from ..models.traffic_policy_phase import TrafficPolicyPhase
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.traffic_policy_backend import TrafficPolicyBackend
    from ..models.traffic_policy_endpoint import TrafficPolicyEndpoint


T = TypeVar("T", bound="TrafficPolicy")


@_attrs_define
class TrafficPolicy:
    """
    Example:
        {'accessUrl': 'https://infer.axisml.io/services/team-vision/resnet-serving/', 'backends': [{'actualPct': 90,
            'ready': True, 'role': 'stable', 'serviceName': 'resnet-serving-v1', 'weight': 90}, {'actualPct': 10, 'ready':
            True, 'role': 'canary', 'serviceName': 'resnet-serving-v2', 'weight': 10}], 'canaryPercent': 10, 'createdAt':
            '2026-06-20T08:00:00Z', 'description': 'Canary traffic split for the ResNet-50 online inference service.',
            'displayName': 'ResNet inference traffic', 'endpoint': {'hostname': 'infer.axisml.io', 'path': '/services/team-
            vision/resnet-serving/'}, 'id': 'd4e5f6a7-8b9c-0d1e-2f3a-4b5c6d7e8f90', 'message': 'Routing 90/10 between stable
            and canary.', 'mode': 'canary', 'name': 'resnet-serving', 'namespace': 'team-vision', 'owner': 'li.wei',
            'ownerId': '3a2b1c0d-4e5f-6789-abcd-ef0123456789', 'phase': 'Ready', 'tenantDisplayName': 'Vision Team',
            'tenantName': 'team-vision', 'updatedAt': '2026-06-28T09:30:00Z'}

    Attributes:
        backends (list[TrafficPolicyBackend]): Member online services and their weights.
        created_at (datetime.datetime): Time the policy was created.
        id (UUID): Stable traffic policy identifier.
        mode (TrafficPolicyMode):
        name (str): Traffic policy name (unique within the tenant).
        namespace (str): Platform tenant namespace the policy belongs to.
        owner (str): Username of the policy owner.
        tenant_name (str): Tenant identifier owning the policy.
        updated_at (datetime.datetime): Time the policy was last updated.
        access_url (str | Unset): Resolved external URL clients call (read-only).
        canary_percent (int | Unset): For canary mode, percent of traffic on the canary backend (stable = 100−p).
        description (str | Unset): Free-text policy description.
        display_name (str | Unset): Human-readable policy label.
        endpoint (TrafficPolicyEndpoint | Unset):  Example: {'hostname': 'infer.axisml.io', 'path': '/services/team-
            vision/resnet-serving/'}.
        message (str | Unset): Human-readable status detail for the current phase.
        owner_id (UUID | Unset): User ID of the policy owner.
        phase (TrafficPolicyPhase | Unset):
        tenant_display_name (str | Unset): Human-readable tenant name.
    """

    backends: list[TrafficPolicyBackend]
    created_at: datetime.datetime
    id: UUID
    mode: TrafficPolicyMode
    name: str
    namespace: str
    owner: str
    tenant_name: str
    updated_at: datetime.datetime
    access_url: str | Unset = UNSET
    canary_percent: int | Unset = UNSET
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    endpoint: TrafficPolicyEndpoint | Unset = UNSET
    message: str | Unset = UNSET
    owner_id: UUID | Unset = UNSET
    phase: TrafficPolicyPhase | Unset = UNSET
    tenant_display_name: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        backends = []
        for backends_item_data in self.backends:
            backends_item = backends_item_data.to_dict()
            backends.append(backends_item)

        created_at = self.created_at.isoformat()

        id = str(self.id)

        mode = self.mode.value

        name = self.name

        namespace = self.namespace

        owner = self.owner

        tenant_name = self.tenant_name

        updated_at = self.updated_at.isoformat()

        access_url = self.access_url

        canary_percent = self.canary_percent

        description = self.description

        display_name = self.display_name

        endpoint: dict[str, Any] | Unset = UNSET
        if not isinstance(self.endpoint, Unset):
            endpoint = self.endpoint.to_dict()

        message = self.message

        owner_id: str | Unset = UNSET
        if not isinstance(self.owner_id, Unset):
            owner_id = str(self.owner_id)

        phase: str | Unset = UNSET
        if not isinstance(self.phase, Unset):
            phase = self.phase.value

        tenant_display_name = self.tenant_display_name

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "backends": backends,
                "createdAt": created_at,
                "id": id,
                "mode": mode,
                "name": name,
                "namespace": namespace,
                "owner": owner,
                "tenantName": tenant_name,
                "updatedAt": updated_at,
            }
        )
        if access_url is not UNSET:
            field_dict["accessUrl"] = access_url
        if canary_percent is not UNSET:
            field_dict["canaryPercent"] = canary_percent
        if description is not UNSET:
            field_dict["description"] = description
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if endpoint is not UNSET:
            field_dict["endpoint"] = endpoint
        if message is not UNSET:
            field_dict["message"] = message
        if owner_id is not UNSET:
            field_dict["ownerId"] = owner_id
        if phase is not UNSET:
            field_dict["phase"] = phase
        if tenant_display_name is not UNSET:
            field_dict["tenantDisplayName"] = tenant_display_name

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.traffic_policy_backend import TrafficPolicyBackend
        from ..models.traffic_policy_endpoint import TrafficPolicyEndpoint

        d = dict(src_dict)
        backends = []
        _backends = d.pop("backends")
        for backends_item_data in _backends:
            backends_item = TrafficPolicyBackend.from_dict(backends_item_data)

            backends.append(backends_item)

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        id = UUID(d.pop("id"))

        mode = TrafficPolicyMode(d.pop("mode"))

        name = d.pop("name")

        namespace = d.pop("namespace")

        owner = d.pop("owner")

        tenant_name = d.pop("tenantName")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

        access_url = d.pop("accessUrl", UNSET)

        canary_percent = d.pop("canaryPercent", UNSET)

        description = d.pop("description", UNSET)

        display_name = d.pop("displayName", UNSET)

        _endpoint = d.pop("endpoint", UNSET)
        endpoint: TrafficPolicyEndpoint | Unset
        if isinstance(_endpoint, Unset):
            endpoint = UNSET
        else:
            endpoint = TrafficPolicyEndpoint.from_dict(_endpoint)

        message = d.pop("message", UNSET)

        _owner_id = d.pop("ownerId", UNSET)
        owner_id: UUID | Unset
        if isinstance(_owner_id, Unset):
            owner_id = UNSET
        else:
            owner_id = UUID(_owner_id)

        _phase = d.pop("phase", UNSET)
        phase: TrafficPolicyPhase | Unset
        if isinstance(_phase, Unset):
            phase = UNSET
        else:
            phase = TrafficPolicyPhase(_phase)

        tenant_display_name = d.pop("tenantDisplayName", UNSET)

        traffic_policy = cls(
            backends=backends,
            created_at=created_at,
            id=id,
            mode=mode,
            name=name,
            namespace=namespace,
            owner=owner,
            tenant_name=tenant_name,
            updated_at=updated_at,
            access_url=access_url,
            canary_percent=canary_percent,
            description=description,
            display_name=display_name,
            endpoint=endpoint,
            message=message,
            owner_id=owner_id,
            phase=phase,
            tenant_display_name=tenant_display_name,
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
