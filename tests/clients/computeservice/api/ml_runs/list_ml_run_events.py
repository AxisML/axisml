from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.compute_service_error import ComputeServiceError
from ...models.event_list import EventList
from ...types import Response


def _get_kwargs(
    namespace: str,
    mlrun: str,
) -> dict[str, Any]:
    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/namespaces/{namespace}/mlruns/{mlrun}/events".format(
            namespace=quote(str(namespace), safe=""),
            mlrun=quote(str(mlrun), safe=""),
        ),
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ComputeServiceError | EventList:
    if response.status_code == 200:
        response_200 = EventList.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = ComputeServiceError.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = ComputeServiceError.from_dict(response.json())

        return response_401

    if response.status_code == 403:
        response_403 = ComputeServiceError.from_dict(response.json())

        return response_403

    if response.status_code == 404:
        response_404 = ComputeServiceError.from_dict(response.json())

        return response_404

    if response.status_code == 409:
        response_409 = ComputeServiceError.from_dict(response.json())

        return response_409

    if response.status_code == 412:
        response_412 = ComputeServiceError.from_dict(response.json())

        return response_412

    if response.status_code == 422:
        response_422 = ComputeServiceError.from_dict(response.json())

        return response_422

    if response.status_code == 503:
        response_503 = ComputeServiceError.from_dict(response.json())

        return response_503

    response_default = ComputeServiceError.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ComputeServiceError | EventList]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    namespace: str,
    mlrun: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ComputeServiceError | EventList]:
    """List events targeted at the MLRun CR

    Args:
        namespace (str):
        mlrun (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComputeServiceError | EventList]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        mlrun=mlrun,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    namespace: str,
    mlrun: str,
    *,
    client: AuthenticatedClient | Client,
) -> ComputeServiceError | EventList | None:
    """List events targeted at the MLRun CR

    Args:
        namespace (str):
        mlrun (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComputeServiceError | EventList
    """

    return sync_detailed(
        namespace=namespace,
        mlrun=mlrun,
        client=client,
    ).parsed


async def asyncio_detailed(
    namespace: str,
    mlrun: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ComputeServiceError | EventList]:
    """List events targeted at the MLRun CR

    Args:
        namespace (str):
        mlrun (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComputeServiceError | EventList]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        mlrun=mlrun,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    namespace: str,
    mlrun: str,
    *,
    client: AuthenticatedClient | Client,
) -> ComputeServiceError | EventList | None:
    """List events targeted at the MLRun CR

    Args:
        namespace (str):
        mlrun (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComputeServiceError | EventList
    """

    return (
        await asyncio_detailed(
            namespace=namespace,
            mlrun=mlrun,
            client=client,
        )
    ).parsed
