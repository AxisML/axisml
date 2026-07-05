from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.metric_point import MetricPoint


T = TypeVar("T", bound="MetricSeries")


@_attrs_define
class MetricSeries:
    """
    Attributes:
        metric (str): Metric name that was queried.
        range_ (str): Query range window (e.g. 1h, 24h).
        series (list[MetricPoint]): Sampled points, oldest first.
        step (str | Unset): Sampling step between points (e.g. 30s).
        unit (str | Unset): Value unit (cores, bytes, percent, req/s, ms).
    """

    metric: str
    range_: str
    series: list[MetricPoint]
    step: str | Unset = UNSET
    unit: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        metric = self.metric

        range_ = self.range_

        series = []
        for series_item_data in self.series:
            series_item = series_item_data.to_dict()
            series.append(series_item)

        step = self.step

        unit = self.unit

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "metric": metric,
                "range": range_,
                "series": series,
            }
        )
        if step is not UNSET:
            field_dict["step"] = step
        if unit is not UNSET:
            field_dict["unit"] = unit

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.metric_point import MetricPoint

        d = dict(src_dict)
        metric = d.pop("metric")

        range_ = d.pop("range")

        series = []
        _series = d.pop("series")
        for series_item_data in _series:
            series_item = MetricPoint.from_dict(series_item_data)

            series.append(series_item)

        step = d.pop("step", UNSET)

        unit = d.pop("unit", UNSET)

        metric_series = cls(
            metric=metric,
            range_=range_,
            series=series,
            step=step,
            unit=unit,
        )

        metric_series.additional_properties = d
        return metric_series

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
