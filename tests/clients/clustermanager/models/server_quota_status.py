from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.server_quota_status_used import ServerQuotaStatusUsed


T = TypeVar("T", bound="ServerQuotaStatus")


@_attrs_define
class ServerQuotaStatus:
    """
    Attributes:
        pool (str): ResourcePool this quota status applies to.
        ready (bool): Whether the ElasticQuota for this pool is provisioned and ready.
        used (ServerQuotaStatusUsed | Unset): Live resource usage from axisml-scheduler via the ElasticQuota.
    """

    pool: str
    ready: bool
    used: ServerQuotaStatusUsed | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        pool = self.pool

        ready = self.ready

        used: dict[str, Any] | Unset = UNSET
        if not isinstance(self.used, Unset):
            used = self.used.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "pool": pool,
                "ready": ready,
            }
        )
        if used is not UNSET:
            field_dict["used"] = used

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.server_quota_status_used import ServerQuotaStatusUsed

        d = dict(src_dict)
        pool = d.pop("pool")

        ready = d.pop("ready")

        _used = d.pop("used", UNSET)
        used: ServerQuotaStatusUsed | Unset
        if isinstance(_used, Unset):
            used = UNSET
        else:
            used = ServerQuotaStatusUsed.from_dict(_used)

        server_quota_status = cls(
            pool=pool,
            ready=ready,
            used=used,
        )

        server_quota_status.additional_properties = d
        return server_quota_status

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
