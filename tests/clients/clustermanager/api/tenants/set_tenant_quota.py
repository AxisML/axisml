from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.cluster_manager_error import ClusterManagerError
from ...models.quota import Quota
from ...models.set_quota_request import SetQuotaRequest
from ...types import Response


def _get_kwargs(
    tenant: str,
    *,
    body: SetQuotaRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/tenants/{tenant}/quotas".format(
            tenant=quote(str(tenant), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ClusterManagerError | Quota:
    if response.status_code == 200:
        response_200 = Quota.from_dict(response.json())

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
) -> Response[ClusterManagerError | Quota]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    tenant: str,
    *,
    client: AuthenticatedClient | Client,
    body: SetQuotaRequest,
) -> Response[ClusterManagerError | Quota]:
    """Create or replace a pool quota

    Args:
        tenant (str):
        body (SetQuotaRequest):  Example: {'pool': 'gpu-a100', 'units': [{'quantity': 4,
            'unitName': 'a100-2x'}]}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClusterManagerError | Quota]
    """

    kwargs = _get_kwargs(
        tenant=tenant,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    tenant: str,
    *,
    client: AuthenticatedClient | Client,
    body: SetQuotaRequest,
) -> ClusterManagerError | Quota | None:
    """Create or replace a pool quota

    Args:
        tenant (str):
        body (SetQuotaRequest):  Example: {'pool': 'gpu-a100', 'units': [{'quantity': 4,
            'unitName': 'a100-2x'}]}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClusterManagerError | Quota
    """

    return sync_detailed(
        tenant=tenant,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    tenant: str,
    *,
    client: AuthenticatedClient | Client,
    body: SetQuotaRequest,
) -> Response[ClusterManagerError | Quota]:
    """Create or replace a pool quota

    Args:
        tenant (str):
        body (SetQuotaRequest):  Example: {'pool': 'gpu-a100', 'units': [{'quantity': 4,
            'unitName': 'a100-2x'}]}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClusterManagerError | Quota]
    """

    kwargs = _get_kwargs(
        tenant=tenant,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    tenant: str,
    *,
    client: AuthenticatedClient | Client,
    body: SetQuotaRequest,
) -> ClusterManagerError | Quota | None:
    """Create or replace a pool quota

    Args:
        tenant (str):
        body (SetQuotaRequest):  Example: {'pool': 'gpu-a100', 'units': [{'quantity': 4,
            'unitName': 'a100-2x'}]}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClusterManagerError | Quota
    """

    return (
        await asyncio_detailed(
            tenant=tenant,
            client=client,
            body=body,
        )
    ).parsed
