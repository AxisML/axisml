from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ml_run_pod_template_subset import MLRunPodTemplateSubset


T = TypeVar("T", bound="MLRunRoleSpec")


@_attrs_define
class MLRunRoleSpec:
    """
    Attributes:
        name (str):
        replicas (int):
        template (MLRunPodTemplateSubset):
        restart_policy (str | Unset):
    """

    name: str
    replicas: int
    template: MLRunPodTemplateSubset
    restart_policy: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        replicas = self.replicas

        template = self.template.to_dict()

        restart_policy = self.restart_policy

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
                "replicas": replicas,
                "template": template,
            }
        )
        if restart_policy is not UNSET:
            field_dict["restartPolicy"] = restart_policy

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ml_run_pod_template_subset import MLRunPodTemplateSubset

        d = dict(src_dict)
        name = d.pop("name")

        replicas = d.pop("replicas")

        template = MLRunPodTemplateSubset.from_dict(d.pop("template"))

        restart_policy = d.pop("restartPolicy", UNSET)

        ml_run_role_spec = cls(
            name=name,
            replicas=replicas,
            template=template,
            restart_policy=restart_policy,
        )

        ml_run_role_spec.additional_properties = d
        return ml_run_role_spec

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
