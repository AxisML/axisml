from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.ml_service import MLService
from ...models.ml_service_create_request import MLServiceCreateRequest
from ...models.problem import Problem
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    body: MLServiceCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    if not isinstance(x_axisml_tenant, Unset):
        headers["X-Axisml-Tenant"] = x_axisml_tenant

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/mlservices",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> MLService | Problem | None:
    if response.status_code == 201:
        response_201 = MLService.from_dict(response.json())

        return response_201

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
) -> Response[MLService | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: MLServiceCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[MLService | Problem]:
    """Deploy a new online inference service

    Args:
        x_axisml_tenant (str | Unset):
        body (MLServiceCreateRequest):  Example: {'backend': {'engine': 'llminference', 'name':
            'kserve'}, 'description': 'Llama3-8B online inference service.', 'displayName': 'Llama3
            chat service', 'env': [{'name': 'MAX_TOKENS', 'value': '4096'}], 'image':
            'registry.axisml.io/serving/vllm:0.6.0', 'modelName': 'llama3-8b', 'modelVersion':
            '1.2.0', 'name': 'llama3-chat', 'poolName': 'gpu-a100', 'ports': [{'name': 'http', 'port':
            8080}], 'replicas': 3, 'route': {'enabled': True, 'path': '/v1/models/llama3-8b'},
            'unitName': 'a100-1x'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[MLService | Problem]
    """

    kwargs = _get_kwargs(
        body=body,
        x_axisml_tenant=x_axisml_tenant,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    body: MLServiceCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> MLService | Problem | None:
    """Deploy a new online inference service

    Args:
        x_axisml_tenant (str | Unset):
        body (MLServiceCreateRequest):  Example: {'backend': {'engine': 'llminference', 'name':
            'kserve'}, 'description': 'Llama3-8B online inference service.', 'displayName': 'Llama3
            chat service', 'env': [{'name': 'MAX_TOKENS', 'value': '4096'}], 'image':
            'registry.axisml.io/serving/vllm:0.6.0', 'modelName': 'llama3-8b', 'modelVersion':
            '1.2.0', 'name': 'llama3-chat', 'poolName': 'gpu-a100', 'ports': [{'name': 'http', 'port':
            8080}], 'replicas': 3, 'route': {'enabled': True, 'path': '/v1/models/llama3-8b'},
            'unitName': 'a100-1x'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        MLService | Problem
    """

    return sync_detailed(
        client=client,
        body=body,
        x_axisml_tenant=x_axisml_tenant,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: MLServiceCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[MLService | Problem]:
    """Deploy a new online inference service

    Args:
        x_axisml_tenant (str | Unset):
        body (MLServiceCreateRequest):  Example: {'backend': {'engine': 'llminference', 'name':
            'kserve'}, 'description': 'Llama3-8B online inference service.', 'displayName': 'Llama3
            chat service', 'env': [{'name': 'MAX_TOKENS', 'value': '4096'}], 'image':
            'registry.axisml.io/serving/vllm:0.6.0', 'modelName': 'llama3-8b', 'modelVersion':
            '1.2.0', 'name': 'llama3-chat', 'poolName': 'gpu-a100', 'ports': [{'name': 'http', 'port':
            8080}], 'replicas': 3, 'route': {'enabled': True, 'path': '/v1/models/llama3-8b'},
            'unitName': 'a100-1x'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[MLService | Problem]
    """

    kwargs = _get_kwargs(
        body=body,
        x_axisml_tenant=x_axisml_tenant,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: MLServiceCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> MLService | Problem | None:
    """Deploy a new online inference service

    Args:
        x_axisml_tenant (str | Unset):
        body (MLServiceCreateRequest):  Example: {'backend': {'engine': 'llminference', 'name':
            'kserve'}, 'description': 'Llama3-8B online inference service.', 'displayName': 'Llama3
            chat service', 'env': [{'name': 'MAX_TOKENS', 'value': '4096'}], 'image':
            'registry.axisml.io/serving/vllm:0.6.0', 'modelName': 'llama3-8b', 'modelVersion':
            '1.2.0', 'name': 'llama3-chat', 'poolName': 'gpu-a100', 'ports': [{'name': 'http', 'port':
            8080}], 'replicas': 3, 'route': {'enabled': True, 'path': '/v1/models/llama3-8b'},
            'unitName': 'a100-1x'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        MLService | Problem
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
            x_axisml_tenant=x_axisml_tenant,
        )
    ).parsed
