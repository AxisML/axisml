from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.problem import Problem
from ...models.resource_unit import ResourceUnit
from ...models.resource_unit_patch_request import ResourceUnitPatchRequest
from ...types import Response


def _get_kwargs(
    pool: str,
    unit: str,
    *,
    body: ResourceUnitPatchRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "patch",
        "url": "/api/v1/resourcepools/{pool}/units/{unit}".format(
            pool=quote(str(pool), safe=""),
            unit=quote(str(unit), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Problem | ResourceUnit | None:
    if response.status_code == 200:
        response_200 = ResourceUnit.from_dict(response.json())

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
    unit: str,
    *,
    client: AuthenticatedClient | Client,
    body: ResourceUnitPatchRequest,
) -> Response[Problem | ResourceUnit]:
    """Update a resource unit

    Args:
        pool (str):
        unit (str):
        body (ResourceUnitPatchRequest):  Example: {'description': 'Updated 2x A100 GPU compute
            unit.', 'limits': {'cpu': '24', 'memory': '192Gi', 'nvidia.com/gpu': '2'}}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ResourceUnit]
    """

    kwargs = _get_kwargs(
        pool=pool,
        unit=unit,
        body=body,
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
    body: ResourceUnitPatchRequest,
) -> Problem | ResourceUnit | None:
    """Update a resource unit

    Args:
        pool (str):
        unit (str):
        body (ResourceUnitPatchRequest):  Example: {'description': 'Updated 2x A100 GPU compute
            unit.', 'limits': {'cpu': '24', 'memory': '192Gi', 'nvidia.com/gpu': '2'}}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ResourceUnit
    """

    return sync_detailed(
        pool=pool,
        unit=unit,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    pool: str,
    unit: str,
    *,
    client: AuthenticatedClient | Client,
    body: ResourceUnitPatchRequest,
) -> Response[Problem | ResourceUnit]:
    """Update a resource unit

    Args:
        pool (str):
        unit (str):
        body (ResourceUnitPatchRequest):  Example: {'description': 'Updated 2x A100 GPU compute
            unit.', 'limits': {'cpu': '24', 'memory': '192Gi', 'nvidia.com/gpu': '2'}}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ResourceUnit]
    """

    kwargs = _get_kwargs(
        pool=pool,
        unit=unit,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    pool: str,
    unit: str,
    *,
    client: AuthenticatedClient | Client,
    body: ResourceUnitPatchRequest,
) -> Problem | ResourceUnit | None:
    """Update a resource unit

    Args:
        pool (str):
        unit (str):
        body (ResourceUnitPatchRequest):  Example: {'description': 'Updated 2x A100 GPU compute
            unit.', 'limits': {'cpu': '24', 'memory': '192Gi', 'nvidia.com/gpu': '2'}}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ResourceUnit
    """

    return (
        await asyncio_detailed(
            pool=pool,
            unit=unit,
            client=client,
            body=body,
        )
    ).parsed
