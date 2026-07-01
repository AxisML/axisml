from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="ArtifactCompleteRequest")


@_attrs_define
class ArtifactCompleteRequest:
    """
    Example:
        {'digest': 'sha256:9b2c1f4e22a74c0e9b1d7f3a2e5c9a108c1f4e222b7a4c0e9b1d7f3a2e5c9a10'}

    Attributes:
        digest (str): Content digest of the pushed artifact, finalizing the two-phase write.
    """

    digest: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        digest = self.digest

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "digest": digest,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        digest = d.pop("digest")

        artifact_complete_request = cls(
            digest=digest,
        )

        artifact_complete_request.additional_properties = d
        return artifact_complete_request

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
