from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.cluster_manager_error import ClusterManagerError
from ...models.pool_metric_series import PoolMetricSeries
from ...types import UNSET, Response, Unset


def _get_kwargs(
    pool: str,
    *,
    tenant: str,
    metric: str,
    range_: str,
    step: str | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["tenant"] = tenant

    params["metric"] = metric

    params["range"] = range_

    params["step"] = step

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/resourcepools/{pool}/metrics".format(
            pool=quote(str(pool), safe=""),
        ),
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ClusterManagerError | PoolMetricSeries:
    if response.status_code == 200:
        response_200 = PoolMetricSeries.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = ClusterManagerError.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = ClusterManagerError.from_dict(response.json())

        return response_401

    if response.status_code == 404:
        response_404 = ClusterManagerError.from_dict(response.json())

        return response_404

    if response.status_code == 409:
        response_409 = ClusterManagerError.from_dict(response.json())

        return response_409

    if response.status_code == 422:
        response_422 = ClusterManagerError.from_dict(response.json())

        return response_422

    if response.status_code == 500:
        response_500 = ClusterManagerError.from_dict(response.json())

        return response_500

    response_default = ClusterManagerError.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ClusterManagerError | PoolMetricSeries]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    pool: str,
    *,
    client: AuthenticatedClient | Client,
    tenant: str,
    metric: str,
    range_: str,
    step: str | Unset = UNSET,
) -> Response[ClusterManagerError | PoolMetricSeries]:
    """Query a tenant's resource-utilisation time series for the pool

    Args:
        pool (str):
        tenant (str):
        metric (str):
        range_ (str):
        step (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClusterManagerError | PoolMetricSeries]
    """

    kwargs = _get_kwargs(
        pool=pool,
        tenant=tenant,
        metric=metric,
        range_=range_,
        step=step,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    pool: str,
    *,
    client: AuthenticatedClient | Client,
    tenant: str,
    metric: str,
    range_: str,
    step: str | Unset = UNSET,
) -> ClusterManagerError | PoolMetricSeries | None:
    """Query a tenant's resource-utilisation time series for the pool

    Args:
        pool (str):
        tenant (str):
        metric (str):
        range_ (str):
        step (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClusterManagerError | PoolMetricSeries
    """

    return sync_detailed(
        pool=pool,
        client=client,
        tenant=tenant,
        metric=metric,
        range_=range_,
        step=step,
    ).parsed


async def asyncio_detailed(
    pool: str,
    *,
    client: AuthenticatedClient | Client,
    tenant: str,
    metric: str,
    range_: str,
    step: str | Unset = UNSET,
) -> Response[ClusterManagerError | PoolMetricSeries]:
    """Query a tenant's resource-utilisation time series for the pool

    Args:
        pool (str):
        tenant (str):
        metric (str):
        range_ (str):
        step (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClusterManagerError | PoolMetricSeries]
    """

    kwargs = _get_kwargs(
        pool=pool,
        tenant=tenant,
        metric=metric,
        range_=range_,
        step=step,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    pool: str,
    *,
    client: AuthenticatedClient | Client,
    tenant: str,
    metric: str,
    range_: str,
    step: str | Unset = UNSET,
) -> ClusterManagerError | PoolMetricSeries | None:
    """Query a tenant's resource-utilisation time series for the pool

    Args:
        pool (str):
        tenant (str):
        metric (str):
        range_ (str):
        step (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClusterManagerError | PoolMetricSeries
    """

    return (
        await asyncio_detailed(
            pool=pool,
            client=client,
            tenant=tenant,
            metric=metric,
            range_=range_,
            step=step,
        )
    ).parsed
