from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.cluster_manager_error import ClusterManagerError
from ...types import Response


def _get_kwargs(
    tenant: str,
    pool: str,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": "/api/v1/tenants/{tenant}/quotas/{pool}".format(
            tenant=quote(str(tenant), safe=""),
            pool=quote(str(pool), safe=""),
        ),
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Any | ClusterManagerError:
    if response.status_code == 204:
        response_204 = cast(Any, None)
        return response_204

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
) -> Response[Any | ClusterManagerError]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    tenant: str,
    pool: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[Any | ClusterManagerError]:
    """Delete a pool quota

    Args:
        tenant (str):
        pool (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | ClusterManagerError]
    """

    kwargs = _get_kwargs(
        tenant=tenant,
        pool=pool,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    tenant: str,
    pool: str,
    *,
    client: AuthenticatedClient | Client,
) -> Any | ClusterManagerError | None:
    """Delete a pool quota

    Args:
        tenant (str):
        pool (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | ClusterManagerError
    """

    return sync_detailed(
        tenant=tenant,
        pool=pool,
        client=client,
    ).parsed


async def asyncio_detailed(
    tenant: str,
    pool: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[Any | ClusterManagerError]:
    """Delete a pool quota

    Args:
        tenant (str):
        pool (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Any | ClusterManagerError]
    """

    kwargs = _get_kwargs(
        tenant=tenant,
        pool=pool,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    tenant: str,
    pool: str,
    *,
    client: AuthenticatedClient | Client,
) -> Any | ClusterManagerError | None:
    """Delete a pool quota

    Args:
        tenant (str):
        pool (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Any | ClusterManagerError
    """

    return (
        await asyncio_detailed(
            tenant=tenant,
            pool=pool,
            client=client,
        )
    ).parsed
