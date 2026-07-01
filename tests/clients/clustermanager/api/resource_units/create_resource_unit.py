from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.cluster_manager_error import ClusterManagerError
from ...models.create_resource_unit_request import CreateResourceUnitRequest
from ...models.resource_unit import ResourceUnit
from ...types import Response


def _get_kwargs(
    pool: str,
    *,
    body: CreateResourceUnitRequest,
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
) -> ClusterManagerError | ResourceUnit:
    if response.status_code == 201:
        response_201 = ResourceUnit.from_dict(response.json())

        return response_201

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
    *,
    client: AuthenticatedClient | Client,
    body: CreateResourceUnitRequest,
) -> Response[ClusterManagerError | ResourceUnit]:
    """Add a unit to the pool

    Args:
        pool (str):
        body (CreateResourceUnitRequest):  Example: {'description': '2× A100 GPU compute unit.',
            'limits': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'}, 'name': 'a100-2x',
            'nodeSelector': {'axisml.io/gpu': 'a100'}, 'requests': {'cpu': '16', 'memory': '128Gi',
            'nvidia.com/gpu': '2'}}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClusterManagerError | ResourceUnit]
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
    body: CreateResourceUnitRequest,
) -> ClusterManagerError | ResourceUnit | None:
    """Add a unit to the pool

    Args:
        pool (str):
        body (CreateResourceUnitRequest):  Example: {'description': '2× A100 GPU compute unit.',
            'limits': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'}, 'name': 'a100-2x',
            'nodeSelector': {'axisml.io/gpu': 'a100'}, 'requests': {'cpu': '16', 'memory': '128Gi',
            'nvidia.com/gpu': '2'}}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClusterManagerError | ResourceUnit
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
    body: CreateResourceUnitRequest,
) -> Response[ClusterManagerError | ResourceUnit]:
    """Add a unit to the pool

    Args:
        pool (str):
        body (CreateResourceUnitRequest):  Example: {'description': '2× A100 GPU compute unit.',
            'limits': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'}, 'name': 'a100-2x',
            'nodeSelector': {'axisml.io/gpu': 'a100'}, 'requests': {'cpu': '16', 'memory': '128Gi',
            'nvidia.com/gpu': '2'}}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClusterManagerError | ResourceUnit]
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
    body: CreateResourceUnitRequest,
) -> ClusterManagerError | ResourceUnit | None:
    """Add a unit to the pool

    Args:
        pool (str):
        body (CreateResourceUnitRequest):  Example: {'description': '2× A100 GPU compute unit.',
            'limits': {'cpu': '16', 'memory': '128Gi', 'nvidia.com/gpu': '2'}, 'name': 'a100-2x',
            'nodeSelector': {'axisml.io/gpu': 'a100'}, 'requests': {'cpu': '16', 'memory': '128Gi',
            'nvidia.com/gpu': '2'}}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClusterManagerError | ResourceUnit
    """

    return (
        await asyncio_detailed(
            pool=pool,
            client=client,
            body=body,
        )
    ).parsed
