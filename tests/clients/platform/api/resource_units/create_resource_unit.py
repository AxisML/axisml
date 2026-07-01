from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.problem import Problem
from ...models.resource_unit import ResourceUnit
from ...models.resource_unit_create_request import ResourceUnitCreateRequest
from ...types import Response


def _get_kwargs(
    pool: str,
    *,
    body: ResourceUnitCreateRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/resourcepools/{pool}/units".format(
            pool=quote(str(pool), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Problem | ResourceUnit | None:
    if response.status_code == 201:
        response_201 = ResourceUnit.from_dict(response.json())

        return response_201

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

    if response.status_code == 409:
        response_409 = Problem.from_dict(response.json())

        return response_409

    if response.status_code == 422:
        response_422 = Problem.from_dict(response.json())

        return response_422

    if response.status_code == 500:
        response_500 = Problem.from_dict(response.json())

        return response_500

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Problem | ResourceUnit]:
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
    body: ResourceUnitCreateRequest,
) -> Response[Problem | ResourceUnit]:
    """Create a resource unit in a pool

    Args:
        pool (str):
        body (ResourceUnitCreateRequest):  Example: {'description': '2x A100 GPU compute unit.',
            'limits': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'}, 'name': 'a100-2x',
            'requests': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'}}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ResourceUnit]
    """

    kwargs = _get_kwargs(
        pool=pool,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    pool: str,
    *,
    client: AuthenticatedClient | Client,
    body: ResourceUnitCreateRequest,
) -> Problem | ResourceUnit | None:
    """Create a resource unit in a pool

    Args:
        pool (str):
        body (ResourceUnitCreateRequest):  Example: {'description': '2x A100 GPU compute unit.',
            'limits': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'}, 'name': 'a100-2x',
            'requests': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'}}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ResourceUnit
    """

    return sync_detailed(
        pool=pool,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    pool: str,
    *,
    client: AuthenticatedClient | Client,
    body: ResourceUnitCreateRequest,
) -> Response[Problem | ResourceUnit]:
    """Create a resource unit in a pool

    Args:
        pool (str):
        body (ResourceUnitCreateRequest):  Example: {'description': '2x A100 GPU compute unit.',
            'limits': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'}, 'name': 'a100-2x',
            'requests': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'}}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ResourceUnit]
    """

    kwargs = _get_kwargs(
        pool=pool,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    pool: str,
    *,
    client: AuthenticatedClient | Client,
    body: ResourceUnitCreateRequest,
) -> Problem | ResourceUnit | None:
    """Create a resource unit in a pool

    Args:
        pool (str):
        body (ResourceUnitCreateRequest):  Example: {'description': '2x A100 GPU compute unit.',
            'limits': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'}, 'name': 'a100-2x',
            'requests': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'}}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ResourceUnit
    """

    return (
        await asyncio_detailed(
            pool=pool,
            client=client,
            body=body,
        )
    ).parsed
