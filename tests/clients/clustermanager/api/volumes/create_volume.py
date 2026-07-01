from http import HTTPStatus
from typing import Any

import httpx

from ...client import AuthenticatedClient, Client
from ...models.cluster_manager_error import ClusterManagerError
from ...models.create_volume_request import CreateVolumeRequest
from ...models.volume import Volume
from ...types import Response


def _get_kwargs(
    *,
    body: CreateVolumeRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/volumes",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ClusterManagerError | Volume:
    if response.status_code == 201:
        response_201 = Volume.from_dict(response.json())

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
) -> Response[ClusterManagerError | Volume]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: CreateVolumeRequest,
) -> Response[ClusterManagerError | Volume]:
    """Materialise a durable volume

    Args:
        body (CreateVolumeRequest):  Example: {'accessModes': ['ReadWriteMany'], 'description':
            'Shared raw datasets directory', 'name': 'shared-datasets', 'namespace': 'team-vision',
            'size': '2Ti', 'storageClass': 'nfs-rwx'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClusterManagerError | Volume]
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
    body: CreateVolumeRequest,
) -> ClusterManagerError | Volume | None:
    """Materialise a durable volume

    Args:
        body (CreateVolumeRequest):  Example: {'accessModes': ['ReadWriteMany'], 'description':
            'Shared raw datasets directory', 'name': 'shared-datasets', 'namespace': 'team-vision',
            'size': '2Ti', 'storageClass': 'nfs-rwx'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClusterManagerError | Volume
    """

    return sync_detailed(
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: CreateVolumeRequest,
) -> Response[ClusterManagerError | Volume]:
    """Materialise a durable volume

    Args:
        body (CreateVolumeRequest):  Example: {'accessModes': ['ReadWriteMany'], 'description':
            'Shared raw datasets directory', 'name': 'shared-datasets', 'namespace': 'team-vision',
            'size': '2Ti', 'storageClass': 'nfs-rwx'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClusterManagerError | Volume]
    """

    kwargs = _get_kwargs(
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: CreateVolumeRequest,
) -> ClusterManagerError | Volume | None:
    """Materialise a durable volume

    Args:
        body (CreateVolumeRequest):  Example: {'accessModes': ['ReadWriteMany'], 'description':
            'Shared raw datasets directory', 'name': 'shared-datasets', 'namespace': 'team-vision',
            'size': '2Ti', 'storageClass': 'nfs-rwx'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClusterManagerError | Volume
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
        )
    ).parsed
