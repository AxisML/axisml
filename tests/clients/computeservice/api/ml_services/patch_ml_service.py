from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.compute_service_error import ComputeServiceError
from ...models.ml_service import MLService
from ...models.ml_service_patch_request import MLServicePatchRequest
from ...types import Response


def _get_kwargs(
    namespace: str,
    mlservice: str,
    *,
    body: MLServicePatchRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "patch",
        "url": "/api/v1/namespaces/{namespace}/mlservices/{mlservice}".format(
            namespace=quote(str(namespace), safe=""),
            mlservice=quote(str(mlservice), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ComputeServiceError | MLService:
    if response.status_code == 200:
        response_200 = MLService.from_dict(response.json())

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
) -> Response[ComputeServiceError | MLService]:
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
    body: MLServicePatchRequest,
) -> Response[ComputeServiceError | MLService]:
    """Patch MLService display fields

    Args:
        namespace (str):
        mlservice (str):
        body (MLServicePatchRequest):  Example: {'description': 'Updated description.',
            'displayName': 'Llama-3 8B inference service (production)'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComputeServiceError | MLService]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        mlservice=mlservice,
        body=body,
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
    body: MLServicePatchRequest,
) -> ComputeServiceError | MLService | None:
    """Patch MLService display fields

    Args:
        namespace (str):
        mlservice (str):
        body (MLServicePatchRequest):  Example: {'description': 'Updated description.',
            'displayName': 'Llama-3 8B inference service (production)'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComputeServiceError | MLService
    """

    return sync_detailed(
        namespace=namespace,
        mlservice=mlservice,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    namespace: str,
    mlservice: str,
    *,
    client: AuthenticatedClient | Client,
    body: MLServicePatchRequest,
) -> Response[ComputeServiceError | MLService]:
    """Patch MLService display fields

    Args:
        namespace (str):
        mlservice (str):
        body (MLServicePatchRequest):  Example: {'description': 'Updated description.',
            'displayName': 'Llama-3 8B inference service (production)'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComputeServiceError | MLService]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        mlservice=mlservice,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    namespace: str,
    mlservice: str,
    *,
    client: AuthenticatedClient | Client,
    body: MLServicePatchRequest,
) -> ComputeServiceError | MLService | None:
    """Patch MLService display fields

    Args:
        namespace (str):
        mlservice (str):
        body (MLServicePatchRequest):  Example: {'description': 'Updated description.',
            'displayName': 'Llama-3 8B inference service (production)'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComputeServiceError | MLService
    """

    return (
        await asyncio_detailed(
            namespace=namespace,
            mlservice=mlservice,
            client=client,
            body=body,
        )
    ).parsed
