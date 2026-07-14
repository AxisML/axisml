from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.compute_service_error import ComputeServiceError
from ...models.traffic_policy import TrafficPolicy
from ...types import Response


def _get_kwargs(
    namespace: str,
    policy: str,
) -> dict[str, Any]:
    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/namespaces/{namespace}/traffic-policies/{policy}/promote".format(
            namespace=quote(str(namespace), safe=""),
            policy=quote(str(policy), safe=""),
        ),
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ComputeServiceError | TrafficPolicy:
    if response.status_code == 202:
        response_202 = TrafficPolicy.from_dict(response.json())

        return response_202

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
) -> Response[ComputeServiceError | TrafficPolicy]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    namespace: str,
    policy: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ComputeServiceError | TrafficPolicy]:
    """Promote the canary to stable (canary mode)

    Args:
        namespace (str):
        policy (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComputeServiceError | TrafficPolicy]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        policy=policy,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    namespace: str,
    policy: str,
    *,
    client: AuthenticatedClient | Client,
) -> ComputeServiceError | TrafficPolicy | None:
    """Promote the canary to stable (canary mode)

    Args:
        namespace (str):
        policy (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComputeServiceError | TrafficPolicy
    """

    return sync_detailed(
        namespace=namespace,
        policy=policy,
        client=client,
    ).parsed


async def asyncio_detailed(
    namespace: str,
    policy: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ComputeServiceError | TrafficPolicy]:
    """Promote the canary to stable (canary mode)

    Args:
        namespace (str):
        policy (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComputeServiceError | TrafficPolicy]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        policy=policy,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    namespace: str,
    policy: str,
    *,
    client: AuthenticatedClient | Client,
) -> ComputeServiceError | TrafficPolicy | None:
    """Promote the canary to stable (canary mode)

    Args:
        namespace (str):
        policy (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComputeServiceError | TrafficPolicy
    """

    return (
        await asyncio_detailed(
            namespace=namespace,
            policy=policy,
            client=client,
        )
    ).parsed
