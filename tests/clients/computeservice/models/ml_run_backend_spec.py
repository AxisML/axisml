from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ml_run_backend_spec_config_type_0 import MLRunBackendSpecConfigType0


T = TypeVar("T", bound="MLRunBackendSpec")


@_attrs_define
class MLRunBackendSpec:
    """
    Attributes:
        engine (str):
        name (str):
        config (MLRunBackendSpecConfigType0 | None | Unset): Embedded Kubernetes resource (free-form JSON).
    """

    engine: str
    name: str
    config: MLRunBackendSpecConfigType0 | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.ml_run_backend_spec_config_type_0 import (
            MLRunBackendSpecConfigType0,
        )

        engine = self.engine

        name = self.name

        config: dict[str, Any] | None | Unset
        if isinstance(self.config, Unset):
            config = UNSET
        elif isinstance(self.config, MLRunBackendSpecConfigType0):
            config = self.config.to_dict()
        else:
            config = self.config

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
        from ..models.ml_run_backend_spec_config_type_0 import (
            MLRunBackendSpecConfigType0,
        )

        d = dict(src_dict)
        engine = d.pop("engine")

        name = d.pop("name")

        def _parse_config(data: object) -> MLRunBackendSpecConfigType0 | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                config_type_0 = MLRunBackendSpecConfigType0.from_dict(data)

                return config_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(MLRunBackendSpecConfigType0 | None | Unset, data)

        config = _parse_config(d.pop("config", UNSET))

        ml_run_backend_spec = cls(
            engine=engine,
            name=name,
            config=config,
        )

        ml_run_backend_spec.additional_properties = d
        return ml_run_backend_spec

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
