from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.get_ml_service_metrics_percentile import GetMLServiceMetricsPercentile
from ...models.metric_series import MetricSeries
from ...models.ml_service_metric_name import MLServiceMetricName
from ...models.problem import Problem
from ...types import UNSET, Response, Unset


def _get_kwargs(
    name: str,
    *,
    metric: MLServiceMetricName,
    range_: str,
    step: str | Unset = UNSET,
    percentile: GetMLServiceMetricsPercentile | Unset = UNSET,
) -> dict[str, Any]:
    params: dict[str, Any] = {}

    json_metric = metric.value
    params["metric"] = json_metric

    params["range"] = range_

    params["step"] = step

    json_percentile: str | Unset = UNSET
    if not isinstance(percentile, Unset):
        json_percentile = percentile.value

    params["percentile"] = json_percentile

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/mlservices/{name}/metrics".format(
            name=quote(str(name), safe=""),
        ),
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> MetricSeries | Problem | None:
    if response.status_code == 200:
        response_200 = MetricSeries.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Problem.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = Problem.from_dict(response.json())

        return response_401

    if response.status_code == 403:
        response_403 = Problem.from_dict(response.json())

        return response_403

    if response.status_code == 404:
        response_404 = Problem.from_dict(response.json())

        return response_404

    if response.status_code == 500:
        response_500 = Problem.from_dict(response.json())

        return response_500

    if response.status_code == 502:
        response_502 = Problem.from_dict(response.json())

        return response_502

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[MetricSeries | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    name: str,
    *,
    client: AuthenticatedClient | Client,
    metric: MLServiceMetricName,
    range_: str,
    step: str | Unset = UNSET,
    percentile: GetMLServiceMetricsPercentile | Unset = UNSET,
) -> Response[MetricSeries | Problem]:
    """Query Prometheus for service-level metrics

    Args:
        name (str):
        metric (MLServiceMetricName):
        range_ (str):
        step (str | Unset):
        percentile (GetMLServiceMetricsPercentile | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[MetricSeries | Problem]
    """

    kwargs = _get_kwargs(
        name=name,
        metric=metric,
        range_=range_,
        step=step,
        percentile=percentile,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    name: str,
    *,
    client: AuthenticatedClient | Client,
    metric: MLServiceMetricName,
    range_: str,
    step: str | Unset = UNSET,
    percentile: GetMLServiceMetricsPercentile | Unset = UNSET,
) -> MetricSeries | Problem | None:
    """Query Prometheus for service-level metrics

    Args:
        name (str):
        metric (MLServiceMetricName):
        range_ (str):
        step (str | Unset):
        percentile (GetMLServiceMetricsPercentile | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        MetricSeries | Problem
    """

    return sync_detailed(
        name=name,
        client=client,
        metric=metric,
        range_=range_,
        step=step,
        percentile=percentile,
    ).parsed


async def asyncio_detailed(
    name: str,
    *,
    client: AuthenticatedClient | Client,
    metric: MLServiceMetricName,
    range_: str,
    step: str | Unset = UNSET,
    percentile: GetMLServiceMetricsPercentile | Unset = UNSET,
) -> Response[MetricSeries | Problem]:
    """Query Prometheus for service-level metrics

    Args:
        name (str):
        metric (MLServiceMetricName):
        range_ (str):
        step (str | Unset):
        percentile (GetMLServiceMetricsPercentile | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[MetricSeries | Problem]
    """

    kwargs = _get_kwargs(
        name=name,
        metric=metric,
        range_=range_,
        step=step,
        percentile=percentile,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    name: str,
    *,
    client: AuthenticatedClient | Client,
    metric: MLServiceMetricName,
    range_: str,
    step: str | Unset = UNSET,
    percentile: GetMLServiceMetricsPercentile | Unset = UNSET,
) -> MetricSeries | Problem | None:
    """Query Prometheus for service-level metrics

    Args:
        name (str):
        metric (MLServiceMetricName):
        range_ (str):
        step (str | Unset):
        percentile (GetMLServiceMetricsPercentile | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        MetricSeries | Problem
    """

    return (
        await asyncio_detailed(
            name=name,
            client=client,
            metric=metric,
            range_=range_,
            step=step,
            percentile=percentile,
        )
    ).parsed
