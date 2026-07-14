from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.image import Image
from ...models.problem import Problem
from ...types import Response


def _get_kwargs(
    tenant: str,
    name: str,
    version: str,
) -> dict[str, Any]:
    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/images/{tenant}/{name}/versions/{version}".format(
            tenant=quote(str(tenant), safe=""),
            name=quote(str(name), safe=""),
            version=quote(str(version), safe=""),
        ),
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Image | Problem | None:
    if response.status_code == 200:
        response_200 = Image.from_dict(response.json())

        return response_200

    if response.status_code == 401:
        response_401 = Problem.from_dict(response.json())

        return response_401

    if response.status_code == 403:
        response_403 = Problem.from_dict(response.json())

        return response_403

    if response.status_code == 404:
        response_404 = Problem.from_dict(response.json())

        return response_404

    if response.status_code == 410:
        response_410 = Problem.from_dict(response.json())

        return response_410

    if response.status_code == 500:
        response_500 = Problem.from_dict(response.json())

        return response_500

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Image | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    tenant: str,
    name: str,
    version: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[Image | Problem]:
    """Get a image artifact

    Args:
        tenant (str):
        name (str):
        version (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Image | Problem]
    """

    kwargs = _get_kwargs(
        tenant=tenant,
        name=name,
        version=version,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    tenant: str,
    name: str,
    version: str,
    *,
    client: AuthenticatedClient | Client,
) -> Image | Problem | None:
    """Get a image artifact

    Args:
        tenant (str):
        name (str):
        version (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Image | Problem
    """

    return sync_detailed(
        tenant=tenant,
        name=name,
        version=version,
        client=client,
    ).parsed


async def asyncio_detailed(
    tenant: str,
    name: str,
    version: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[Image | Problem]:
    """Get a image artifact

    Args:
        tenant (str):
        name (str):
        version (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Image | Problem]
    """

    kwargs = _get_kwargs(
        tenant=tenant,
        name=name,
        version=version,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    tenant: str,
    name: str,
    version: str,
    *,
    client: AuthenticatedClient | Client,
) -> Image | Problem | None:
    """Get a image artifact

    Args:
        tenant (str):
        name (str):
        version (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Image | Problem
    """

    return (
        await asyncio_detailed(
            tenant=tenant,
            name=name,
            version=version,
            client=client,
        )
    ).parsed
