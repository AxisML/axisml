from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.server_quota_resources_max import ServerQuotaResourcesMax
    from ..models.server_quota_resources_min import ServerQuotaResourcesMin


T = TypeVar("T", bound="ServerQuotaResources")


@_attrs_define
class ServerQuotaResources:
    """
    Attributes:
        max_ (ServerQuotaResourcesMax): ElasticQuota maximum resources.
        min_ (ServerQuotaResourcesMin | Unset): ElasticQuota minimum resources.
    """

    max_: ServerQuotaResourcesMax
    min_: ServerQuotaResourcesMin | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        max_ = self.max_.to_dict()

        min_: dict[str, Any] | Unset = UNSET
        if not isinstance(self.min_, Unset):
            min_ = self.min_.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "max": max_,
            }
        )
        if min_ is not UNSET:
            field_dict["min"] = min_

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.server_quota_resources_max import ServerQuotaResourcesMax
        from ..models.server_quota_resources_min import ServerQuotaResourcesMin

        d = dict(src_dict)
        max_ = ServerQuotaResourcesMax.from_dict(d.pop("max"))

        _min_ = d.pop("min", UNSET)
        min_: ServerQuotaResourcesMin | Unset
        if isinstance(_min_, Unset):
            min_ = UNSET
        else:
            min_ = ServerQuotaResourcesMin.from_dict(_min_)

        server_quota_resources = cls(
            max_=max_,
            min_=min_,
        )

        server_quota_resources.additional_properties = d
        return server_quota_resources

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
