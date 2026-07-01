from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.ml_service_pod_template import MLServicePodTemplate


T = TypeVar("T", bound="MLServiceRoleSpec")


@_attrs_define
class MLServiceRoleSpec:
    """
    Attributes:
        name (str):
        replicas (int):
        template (MLServicePodTemplate):
    """

    name: str
    replicas: int
    template: MLServicePodTemplate
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        replicas = self.replicas

        template = self.template.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
                "replicas": replicas,
                "template": template,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ml_service_pod_template import MLServicePodTemplate

        d = dict(src_dict)
        name = d.pop("name")

        replicas = d.pop("replicas")

        template = MLServicePodTemplate.from_dict(d.pop("template"))

        ml_service_role_spec = cls(
            name=name,
            replicas=replicas,
            template=template,
        )

        ml_service_role_spec.additional_properties = d
        return ml_service_role_spec

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
