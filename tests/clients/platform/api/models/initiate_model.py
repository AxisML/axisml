from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.model_initiate_request import ModelInitiateRequest
from ...models.model_initiate_response import ModelInitiateResponse
from ...models.problem import Problem
from ...types import Response


def _get_kwargs(
    tenant: str,
    name: str,
    *,
    body: ModelInitiateRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/models/{tenant}/{name}/versions".format(
            tenant=quote(str(tenant), safe=""),
            name=quote(str(name), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ModelInitiateResponse | Problem | None:
    if response.status_code == 202:
        response_202 = ModelInitiateResponse.from_dict(response.json())

        return response_202

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

    if response.status_code == 409:
        response_409 = Problem.from_dict(response.json())

        return response_409

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
) -> Response[ModelInitiateResponse | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    tenant: str,
    name: str,
    *,
    client: AuthenticatedClient | Client,
    body: ModelInitiateRequest,
) -> Response[ModelInitiateResponse | Problem]:
    """Initiate a model version upload

    Args:
        tenant (str):
        name (str):
        body (ModelInitiateRequest):  Example: {'description': 'ResNet-50 weights fine-tuned on
            ImageNet.', 'displayName': 'ResNet-50 v1.4.0', 'source': 'webUpload', 'version': '1.4.0'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ModelInitiateResponse | Problem]
    """

    kwargs = _get_kwargs(
        tenant=tenant,
        name=name,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    tenant: str,
    name: str,
    *,
    client: AuthenticatedClient | Client,
    body: ModelInitiateRequest,
) -> ModelInitiateResponse | Problem | None:
    """Initiate a model version upload

    Args:
        tenant (str):
        name (str):
        body (ModelInitiateRequest):  Example: {'description': 'ResNet-50 weights fine-tuned on
            ImageNet.', 'displayName': 'ResNet-50 v1.4.0', 'source': 'webUpload', 'version': '1.4.0'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ModelInitiateResponse | Problem
    """

    return sync_detailed(
        tenant=tenant,
        name=name,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    tenant: str,
    name: str,
    *,
    client: AuthenticatedClient | Client,
    body: ModelInitiateRequest,
) -> Response[ModelInitiateResponse | Problem]:
    """Initiate a model version upload

    Args:
        tenant (str):
        name (str):
        body (ModelInitiateRequest):  Example: {'description': 'ResNet-50 weights fine-tuned on
            ImageNet.', 'displayName': 'ResNet-50 v1.4.0', 'source': 'webUpload', 'version': '1.4.0'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ModelInitiateResponse | Problem]
    """

    kwargs = _get_kwargs(
        tenant=tenant,
        name=name,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    tenant: str,
    name: str,
    *,
    client: AuthenticatedClient | Client,
    body: ModelInitiateRequest,
) -> ModelInitiateResponse | Problem | None:
    """Initiate a model version upload

    Args:
        tenant (str):
        name (str):
        body (ModelInitiateRequest):  Example: {'description': 'ResNet-50 weights fine-tuned on
            ImageNet.', 'displayName': 'ResNet-50 v1.4.0', 'source': 'webUpload', 'version': '1.4.0'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ModelInitiateResponse | Problem
    """

    return (
        await asyncio_detailed(
            tenant=tenant,
            name=name,
            client=client,
            body=body,
        )
    ).parsed
