from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.artifact_definition import ArtifactDefinition
from ...models.artifact_definition_create_request import ArtifactDefinitionCreateRequest
from ...models.problem import Problem
from ...types import Response


def _get_kwargs(
    tenant: str,
    name: str,
    *,
    body: ArtifactDefinitionCreateRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/images/{tenant}/{name}".format(
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
) -> ArtifactDefinition | Problem | None:
    if response.status_code == 201:
        response_201 = ArtifactDefinition.from_dict(response.json())

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
) -> Response[ArtifactDefinition | Problem]:
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
    body: ArtifactDefinitionCreateRequest,
) -> Response[ArtifactDefinition | Problem]:
    """Create a image definition

    Args:
        tenant (str):
        name (str):
        body (ArtifactDefinitionCreateRequest):  Example: {'description': 'ResNet-50 image-
            classification model.', 'displayName': 'ResNet-50', 'labels': {'team': 'vision'}, 'name':
            'resnet50'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ArtifactDefinition | Problem]
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
    body: ArtifactDefinitionCreateRequest,
) -> ArtifactDefinition | Problem | None:
    """Create a image definition

    Args:
        tenant (str):
        name (str):
        body (ArtifactDefinitionCreateRequest):  Example: {'description': 'ResNet-50 image-
            classification model.', 'displayName': 'ResNet-50', 'labels': {'team': 'vision'}, 'name':
            'resnet50'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ArtifactDefinition | Problem
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
    body: ArtifactDefinitionCreateRequest,
) -> Response[ArtifactDefinition | Problem]:
    """Create a image definition

    Args:
        tenant (str):
        name (str):
        body (ArtifactDefinitionCreateRequest):  Example: {'description': 'ResNet-50 image-
            classification model.', 'displayName': 'ResNet-50', 'labels': {'team': 'vision'}, 'name':
            'resnet50'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ArtifactDefinition | Problem]
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
    body: ArtifactDefinitionCreateRequest,
) -> ArtifactDefinition | Problem | None:
    """Create a image definition

    Args:
        tenant (str):
        name (str):
        body (ArtifactDefinitionCreateRequest):  Example: {'description': 'ResNet-50 image-
            classification model.', 'displayName': 'ResNet-50', 'labels': {'team': 'vision'}, 'name':
            'resnet50'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ArtifactDefinition | Problem
    """

    return (
        await asyncio_detailed(
            tenant=tenant,
            name=name,
            client=client,
            body=body,
        )
    ).parsed
