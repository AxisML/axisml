from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.compute_service_error import ComputeServiceError
from ...models.metric_series import MetricSeries
from ...types import UNSET, Response, Unset


def _get_kwargs(
    namespace: str,
    mlrun: str,
    *,
    metric: str,
    range_: str,
    step: str | Unset = UNSET,
) -> dict[str, Any]:
    params: dict[str, Any] = {}

    params["metric"] = metric

    params["range"] = range_

    params["step"] = step

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/namespaces/{namespace}/mlruns/{mlrun}/metrics".format(
            namespace=quote(str(namespace), safe=""),
            mlrun=quote(str(mlrun), safe=""),
        ),
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ComputeServiceError | MetricSeries:
    if response.status_code == 200:
        response_200 = MetricSeries.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = ComputeServiceError.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = ComputeServiceError.from_dict(response.json())

        return response_401

    if response.status_code == 403:
        response_403 = ComputeServiceError.from_dict(response.json())

        return response_403

    if response.status_code == 404:
        response_404 = ComputeServiceError.from_dict(response.json())

        return response_404

    if response.status_code == 409:
        response_409 = ComputeServiceError.from_dict(response.json())

        return response_409

    if response.status_code == 412:
        response_412 = ComputeServiceError.from_dict(response.json())

        return response_412

    if response.status_code == 422:
        response_422 = ComputeServiceError.from_dict(response.json())

        return response_422

    if response.status_code == 503:
        response_503 = ComputeServiceError.from_dict(response.json())

        return response_503

    response_default = ComputeServiceError.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ComputeServiceError | MetricSeries]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    namespace: str,
    mlrun: str,
    *,
    client: AuthenticatedClient | Client,
    metric: str,
    range_: str,
    step: str | Unset = UNSET,
) -> Response[ComputeServiceError | MetricSeries]:
    """Query a resource metric time series for the MLRun

    Args:
        namespace (str):
        mlrun (str):
        metric (str):
        range_ (str):
        step (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComputeServiceError | MetricSeries]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        mlrun=mlrun,
        metric=metric,
        range_=range_,
        step=step,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    namespace: str,
    mlrun: str,
    *,
    client: AuthenticatedClient | Client,
    metric: str,
    range_: str,
    step: str | Unset = UNSET,
) -> ComputeServiceError | MetricSeries | None:
    """Query a resource metric time series for the MLRun

    Args:
        namespace (str):
        mlrun (str):
        metric (str):
        range_ (str):
        step (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComputeServiceError | MetricSeries
    """

    return sync_detailed(
        namespace=namespace,
        mlrun=mlrun,
        client=client,
        metric=metric,
        range_=range_,
        step=step,
    ).parsed


async def asyncio_detailed(
    namespace: str,
    mlrun: str,
    *,
    client: AuthenticatedClient | Client,
    metric: str,
    range_: str,
    step: str | Unset = UNSET,
) -> Response[ComputeServiceError | MetricSeries]:
    """Query a resource metric time series for the MLRun

    Args:
        namespace (str):
        mlrun (str):
        metric (str):
        range_ (str):
        step (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComputeServiceError | MetricSeries]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        mlrun=mlrun,
        metric=metric,
        range_=range_,
        step=step,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    namespace: str,
    mlrun: str,
    *,
    client: AuthenticatedClient | Client,
    metric: str,
    range_: str,
    step: str | Unset = UNSET,
) -> ComputeServiceError | MetricSeries | None:
    """Query a resource metric time series for the MLRun

    Args:
        namespace (str):
        mlrun (str):
        metric (str):
        range_ (str):
        step (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComputeServiceError | MetricSeries
    """

    return (
        await asyncio_detailed(
            namespace=namespace,
            mlrun=mlrun,
            client=client,
            metric=metric,
            range_=range_,
            step=step,
        )
    ).parsed
