from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.compute_service_error import ComputeServiceError
from ...models.ml_service_phase import MLServicePhase
from ...types import Response


def _get_kwargs(
    namespace: str,
    mlservice: str,
) -> dict[str, Any]:
    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/namespaces/{namespace}/mlservices/{mlservice}/phase".format(
            namespace=quote(str(namespace), safe=""),
            mlservice=quote(str(mlservice), safe=""),
        ),
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ComputeServiceError | MLServicePhase:
    if response.status_code == 200:
        response_200 = MLServicePhase.from_dict(response.json())

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
) -> Response[ComputeServiceError | MLServicePhase]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    namespace: str,
    mlservice: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ComputeServiceError | MLServicePhase]:
    """Get MLService phase

    Args:
        namespace (str):
        mlservice (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComputeServiceError | MLServicePhase]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        mlservice=mlservice,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    namespace: str,
    mlservice: str,
    *,
    client: AuthenticatedClient | Client,
) -> ComputeServiceError | MLServicePhase | None:
    """Get MLService phase

    Args:
        namespace (str):
        mlservice (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComputeServiceError | MLServicePhase
    """

    return sync_detailed(
        namespace=namespace,
        mlservice=mlservice,
        client=client,
    ).parsed


async def asyncio_detailed(
    namespace: str,
    mlservice: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[ComputeServiceError | MLServicePhase]:
    """Get MLService phase

    Args:
        namespace (str):
        mlservice (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComputeServiceError | MLServicePhase]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        mlservice=mlservice,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    namespace: str,
    mlservice: str,
    *,
    client: AuthenticatedClient | Client,
) -> ComputeServiceError | MLServicePhase | None:
    """Get MLService phase

    Args:
        namespace (str):
        mlservice (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComputeServiceError | MLServicePhase
    """

    return (
        await asyncio_detailed(
            namespace=namespace,
            mlservice=mlservice,
            client=client,
        )
    ).parsed
