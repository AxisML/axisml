from http import HTTPStatus
from typing import Any

import httpx

from ...client import AuthenticatedClient, Client
from ...models.cluster_manager_error import ClusterManagerError
from ...models.resource_pool_list import ResourcePoolList
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    label_selector: str | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["labelSelector"] = label_selector

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/resourcepools",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ClusterManagerError | ResourcePoolList:
    if response.status_code == 200:
        response_200 = ResourcePoolList.from_dict(response.json())

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
) -> Response[ClusterManagerError | ResourcePoolList]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    label_selector: str | Unset = UNSET,
) -> Response[ClusterManagerError | ResourcePoolList]:
    """List ResourcePools

    Args:
        label_selector (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClusterManagerError | ResourcePoolList]
    """

    kwargs = _get_kwargs(
        label_selector=label_selector,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    label_selector: str | Unset = UNSET,
) -> ClusterManagerError | ResourcePoolList | None:
    """List ResourcePools

    Args:
        label_selector (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClusterManagerError | ResourcePoolList
    """

    return sync_detailed(
        client=client,
        label_selector=label_selector,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    label_selector: str | Unset = UNSET,
) -> Response[ClusterManagerError | ResourcePoolList]:
    """List ResourcePools

    Args:
        label_selector (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClusterManagerError | ResourcePoolList]
    """

    kwargs = _get_kwargs(
        label_selector=label_selector,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    label_selector: str | Unset = UNSET,
) -> ClusterManagerError | ResourcePoolList | None:
    """List ResourcePools

    Args:
        label_selector (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClusterManagerError | ResourcePoolList
    """

    return (
        await asyncio_detailed(
            client=client,
            label_selector=label_selector,
        )
    ).parsed
