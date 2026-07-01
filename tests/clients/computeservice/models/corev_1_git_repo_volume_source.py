from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Corev1GitRepoVolumeSource")


@_attrs_define
class Corev1GitRepoVolumeSource:
    """
    Attributes:
        repository (str):
        directory (str | Unset):
        revision (str | Unset):
    """

    repository: str
    directory: str | Unset = UNSET
    revision: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        repository = self.repository

        directory = self.directory

        revision = self.revision

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "repository": repository,
            }
        )
        if directory is not UNSET:
            field_dict["directory"] = directory
        if revision is not UNSET:
            field_dict["revision"] = revision

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        repository = d.pop("repository")

        directory = d.pop("directory", UNSET)

        revision = d.pop("revision", UNSET)

        corev_1_git_repo_volume_source = cls(
            repository=repository,
            directory=directory,
            revision=revision,
        )

        corev_1_git_repo_volume_source.additional_properties = d
        return corev_1_git_repo_volume_source

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
