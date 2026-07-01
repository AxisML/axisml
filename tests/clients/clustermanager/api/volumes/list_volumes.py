from http import HTTPStatus
from typing import Any

import httpx

from ...client import AuthenticatedClient, Client
from ...models.cluster_manager_error import ClusterManagerError
from ...models.volume_list import VolumeList
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    namespace: str | Unset = UNSET,
    label_selector: str | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["namespace"] = namespace

    params["labelSelector"] = label_selector

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/volumes",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ClusterManagerError | VolumeList:
    if response.status_code == 200:
        response_200 = VolumeList.from_dict(response.json())

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
) -> Response[ClusterManagerError | VolumeList]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    namespace: str | Unset = UNSET,
    label_selector: str | Unset = UNSET,
) -> Response[ClusterManagerError | VolumeList]:
    """List durable volumes

    Args:
        namespace (str | Unset):
        label_selector (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClusterManagerError | VolumeList]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        label_selector=label_selector,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    namespace: str | Unset = UNSET,
    label_selector: str | Unset = UNSET,
) -> ClusterManagerError | VolumeList | None:
    """List durable volumes

    Args:
        namespace (str | Unset):
        label_selector (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClusterManagerError | VolumeList
    """

    return sync_detailed(
        client=client,
        namespace=namespace,
        label_selector=label_selector,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    namespace: str | Unset = UNSET,
    label_selector: str | Unset = UNSET,
) -> Response[ClusterManagerError | VolumeList]:
    """List durable volumes

    Args:
        namespace (str | Unset):
        label_selector (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ClusterManagerError | VolumeList]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        label_selector=label_selector,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    namespace: str | Unset = UNSET,
    label_selector: str | Unset = UNSET,
) -> ClusterManagerError | VolumeList | None:
    """List durable volumes

    Args:
        namespace (str | Unset):
        label_selector (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ClusterManagerError | VolumeList
    """

    return (
        await asyncio_detailed(
            client=client,
            namespace=namespace,
            label_selector=label_selector,
        )
    ).parsed
