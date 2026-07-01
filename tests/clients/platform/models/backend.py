from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.backend_name import BackendName
from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.backend_config import BackendConfig


T = TypeVar("T", bound="Backend")


@_attrs_define
class Backend:
    """
    Example:
        {'engine': 'pytorchjob', 'name': 'native'}

    Attributes:
        engine (str): Engine within the backend (e.g. job, pytorchjob, deployment).
        name (BackendName): Compute backend that runs the workload (native, kubeflow-trainer, kserve, custom).
        config (BackendConfig | Unset): Backend-specific free-form config (e.g. the target GVK for the custom backend).
    """

    engine: str
    name: BackendName
    config: BackendConfig | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        engine = self.engine

        name = self.name.value

        config: dict[str, Any] | Unset = UNSET
        if not isinstance(self.config, Unset):
            config = self.config.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "engine": engine,
                "name": name,
            }
        )
        if config is not UNSET:
            field_dict["config"] = config

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.backend_config import BackendConfig

        d = dict(src_dict)
        engine = d.pop("engine")

        name = BackendName(d.pop("name"))

        _config = d.pop("config", UNSET)
        config: BackendConfig | Unset
        if isinstance(_config, Unset):
            config = UNSET
        else:
            config = BackendConfig.from_dict(_config)

        backend = cls(
            engine=engine,
            name=name,
            config=config,
        )

        backend.additional_properties = d
        return backend

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
