from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.cluster_manager_error import ClusterManagerError
from ...models.patch_volume_request import PatchVolumeRequest
from ...models.volume import Volume
from ...types import Response


def _get_kwargs(
    namespace: str,
    name: str,
    *,
    body: PatchVolumeRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "patch",
        "url": "/api/v1/volumes/{namespace}/{name}".format(
            namespace=quote(str(namespace), safe=""),
            name=quote(str(name), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ClusterManagerError | Volume:
    if response.status_code == 200:
        response_200 = Volume.from_dict(response.json())

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
) -> Response[ClusterManagerError | Volume]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    namespace: str,
    name: str,
    *,
    client: AuthenticatedClient | Client,
    body: PatchVolumeRequest,
) -> Response[ClusterManagerError | Volume]:
    """Expand or relabel a durable volume

    Args:
        namespace (str):
        name (str):
        body (PatchVolumeRequest):  Example: {'description': 'Shared raw datasets directory
            (expanded)', 'size': '4Ti'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClusterManagerError | Volume]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        name=name,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    namespace: str,
    name: str,
    *,
    client: AuthenticatedClient | Client,
    body: PatchVolumeRequest,
) -> ClusterManagerError | Volume | None:
    """Expand or relabel a durable volume

    Args:
        namespace (str):
        name (str):
        body (PatchVolumeRequest):  Example: {'description': 'Shared raw datasets directory
            (expanded)', 'size': '4Ti'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClusterManagerError | Volume
    """

    return sync_detailed(
        namespace=namespace,
        name=name,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    namespace: str,
    name: str,
    *,
    client: AuthenticatedClient | Client,
    body: PatchVolumeRequest,
) -> Response[ClusterManagerError | Volume]:
    """Expand or relabel a durable volume

    Args:
        namespace (str):
        name (str):
        body (PatchVolumeRequest):  Example: {'description': 'Shared raw datasets directory
            (expanded)', 'size': '4Ti'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClusterManagerError | Volume]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        name=name,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    namespace: str,
    name: str,
    *,
    client: AuthenticatedClient | Client,
    body: PatchVolumeRequest,
) -> ClusterManagerError | Volume | None:
    """Expand or relabel a durable volume

    Args:
        namespace (str):
        name (str):
        body (PatchVolumeRequest):  Example: {'description': 'Shared raw datasets directory
            (expanded)', 'size': '4Ti'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClusterManagerError | Volume
    """

    return (
        await asyncio_detailed(
            namespace=namespace,
            name=name,
            client=client,
            body=body,
        )
    ).parsed
