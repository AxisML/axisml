from http import HTTPStatus
from typing import Any

import httpx

from ...client import AuthenticatedClient, Client
from ...models.cluster_manager_error import ClusterManagerError
from ...models.create_tenant_request import CreateTenantRequest
from ...models.tenant import Tenant
from ...types import Response


def _get_kwargs(
    *,
    body: CreateTenantRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/tenants",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ClusterManagerError | Tenant:
    if response.status_code == 201:
        response_201 = Tenant.from_dict(response.json())

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
) -> Response[ClusterManagerError | Tenant]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: CreateTenantRequest,
) -> Response[ClusterManagerError | Tenant]:
    """Create a Tenant

    Args:
        body (CreateTenantRequest):  Example: {'labels': {'displayName': 'Vision Team'}, 'name':
            'team-vision', 'namespace': {'name': 'team-vision'}, 'quotas': [{'pool': 'gpu-a100',
            'units': [{'quantity': 4, 'unitName': 'a100-2x'}]}]}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClusterManagerError | Tenant]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    body: CreateTenantRequest,
) -> ClusterManagerError | Tenant | None:
    """Create a Tenant

    Args:
        body (CreateTenantRequest):  Example: {'labels': {'displayName': 'Vision Team'}, 'name':
            'team-vision', 'namespace': {'name': 'team-vision'}, 'quotas': [{'pool': 'gpu-a100',
            'units': [{'quantity': 4, 'unitName': 'a100-2x'}]}]}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClusterManagerError | Tenant
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: CreateTenantRequest,
) -> Response[ClusterManagerError | Tenant]:
    """Create a Tenant

    Args:
        body (CreateTenantRequest):  Example: {'labels': {'displayName': 'Vision Team'}, 'name':
            'team-vision', 'namespace': {'name': 'team-vision'}, 'quotas': [{'pool': 'gpu-a100',
            'units': [{'quantity': 4, 'unitName': 'a100-2x'}]}]}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClusterManagerError | Tenant]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: CreateTenantRequest,
) -> ClusterManagerError | Tenant | None:
    """Create a Tenant

    Args:
        body (CreateTenantRequest):  Example: {'labels': {'displayName': 'Vision Team'}, 'name':
            'team-vision', 'namespace': {'name': 'team-vision'}, 'quotas': [{'pool': 'gpu-a100',
            'units': [{'quantity': 4, 'unitName': 'a100-2x'}]}]}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClusterManagerError | Tenant
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
