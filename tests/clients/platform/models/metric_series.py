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
    Example:
        {'metric': 'qps', 'percentile': 'p95', 'range': '1h', 'series': [{'timestamp': '2026-06-28T09:00:00Z', 'value':
            12.5}, {'timestamp': '2026-06-28T09:30:00Z', 'value': 18.3}, {'timestamp': '2026-06-28T09:25:00Z', 'value':
            15.1}], 'step': '1m', 'unit': 'req/s'}

    Attributes:
        metric (str): Metric name (e.g. qps, latency).
        range_ (str): Query time range (e.g. 1h).
        series (list[MetricPoint]): Ordered (timestamp, value) samples.
        percentile (str | Unset): Percentile the series represents (e.g. p95).
        step (str | Unset): Sampling step between points (e.g. 1m).
        unit (str | Unset): Value unit (e.g. ms, req/s).
    """

    metric: str
    range_: str
    series: list[MetricPoint]
    percentile: str | Unset = UNSET
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

        percentile = self.percentile

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
        if percentile is not UNSET:
            field_dict["percentile"] = percentile
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

        percentile = d.pop("percentile", UNSET)

        step = d.pop("step", UNSET)

        unit = d.pop("unit", UNSET)

        metric_series = cls(
            metric=metric,
            range_=range_,
            series=series,
            percentile=percentile,
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
