from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.cluster_manager_error import ClusterManagerError
from ...models.resource_unit import ResourceUnit
from ...types import Response


def _get_kwargs(
    pool: str,
    unit: str,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/resourcepools/{pool}/units/{unit}".format(
            pool=quote(str(pool), safe=""),
            unit=quote(str(unit), safe=""),
        ),
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ClusterManagerError | ResourceUnit:
    if response.status_code == 200:
        response_200 = ResourceUnit.from_dict(response.json())

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
) -> Response[ClusterManagerError | ResourceUnit]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    pool: str,
    unit: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ClusterManagerError | ResourceUnit]:
    """Get unit

    Args:
        pool (str):
        unit (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClusterManagerError | ResourceUnit]
    """

    kwargs = _get_kwargs(
        pool=pool,
        unit=unit,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    pool: str,
    unit: str,
    *,
    client: AuthenticatedClient | Client,
) -> ClusterManagerError | ResourceUnit | None:
    """Get unit

    Args:
        pool (str):
        unit (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClusterManagerError | ResourceUnit
    """

    return sync_detailed(
        pool=pool,
        unit=unit,
        client=client,
    ).parsed


async def asyncio_detailed(
    pool: str,
    unit: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ClusterManagerError | ResourceUnit]:
    """Get unit

    Args:
        pool (str):
        unit (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClusterManagerError | ResourceUnit]
    """

    kwargs = _get_kwargs(
        pool=pool,
        unit=unit,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    pool: str,
    unit: str,
    *,
    client: AuthenticatedClient | Client,
) -> ClusterManagerError | ResourceUnit | None:
    """Get unit

    Args:
        pool (str):
        unit (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClusterManagerError | ResourceUnit
    """

    return (
        await asyncio_detailed(
            pool=pool,
            unit=unit,
            client=client,
        )
    ).parsed
