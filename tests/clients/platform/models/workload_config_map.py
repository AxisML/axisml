from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.workload_config_map_data import WorkloadConfigMapData


T = TypeVar("T", bound="WorkloadConfigMap")


@_attrs_define
class WorkloadConfigMap:
    r"""
    Example:
        {'data': {'trainer.yaml': 'epochs: 90\nbatchSize: 256\n'}, 'name': 'resnet-training-config'}

    Attributes:
        name (str): DNS-1123 ConfigMap name in the workload namespace.
        data (WorkloadConfigMapData | Unset): UTF-8 configuration entries keyed by file or environment-variable name.
    """

    name: str
    data: WorkloadConfigMapData | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        data: dict[str, Any] | Unset = UNSET
        if not isinstance(self.data, Unset):
            data = self.data.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
            }
        )
        if data is not UNSET:
            field_dict["data"] = data

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.workload_config_map_data import WorkloadConfigMapData

        d = dict(src_dict)
        name = d.pop("name")

        _data = d.pop("data", UNSET)
        data: WorkloadConfigMapData | Unset
        if isinstance(_data, Unset):
            data = UNSET
        else:
            data = WorkloadConfigMapData.from_dict(_data)

        workload_config_map = cls(
            name=name,
            data=data,
        )

        workload_config_map.additional_properties = d
        return workload_config_map

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
