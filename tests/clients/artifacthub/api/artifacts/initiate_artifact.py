from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.artifact_hub_error import ArtifactHubError
from ...models.artifact_initiate_request import ArtifactInitiateRequest
from ...models.artifact_initiate_response import ArtifactInitiateResponse
from ...types import Response


def _get_kwargs(
    namespace: str,
    name: str,
    *,
    body: ArtifactInitiateRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/namespaces/{namespace}/artifacts/{name}".format(
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
) -> ArtifactHubError | ArtifactInitiateResponse:
    if response.status_code == 201:
        response_201 = ArtifactInitiateResponse.from_dict(response.json())

        return response_201

    if response.status_code == 400:
        response_400 = ArtifactHubError.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = ArtifactHubError.from_dict(response.json())

        return response_401

    if response.status_code == 403:
        response_403 = ArtifactHubError.from_dict(response.json())

        return response_403

    if response.status_code == 404:
        response_404 = ArtifactHubError.from_dict(response.json())

        return response_404

    if response.status_code == 409:
        response_409 = ArtifactHubError.from_dict(response.json())

        return response_409

    if response.status_code == 410:
        response_410 = ArtifactHubError.from_dict(response.json())

        return response_410

    if response.status_code == 412:
        response_412 = ArtifactHubError.from_dict(response.json())

        return response_412

    if response.status_code == 503:
        response_503 = ArtifactHubError.from_dict(response.json())

        return response_503

    response_default = ArtifactHubError.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ArtifactHubError | ArtifactInitiateResponse]:
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
    body: ArtifactInitiateRequest,
) -> Response[ArtifactHubError | ArtifactInitiateResponse]:
    """Initiate an artifact version (two-phase write step 1)

    Args:
        namespace (str):
        name (str):
        body (ArtifactInitiateRequest):  Example: {'annotations': {'git-commit': '8c1f4e2'},
            'description': 'ResNet-50 image-classification model pretrained on ImageNet.',
            'displayName': 'ResNet-50', 'labels': {'stage': 'production', 'team': 'vision'}, 'source':
            'webUpload', 'spec': {'format': 'safetensors', 'framework': 'pytorch', 'parameters':
            '25.6M', 'task': 'image-classification'}, 'version': '1.4.0', 'visibility': 'tenant'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ArtifactHubError | ArtifactInitiateResponse]
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
    body: ArtifactInitiateRequest,
) -> ArtifactHubError | ArtifactInitiateResponse | None:
    """Initiate an artifact version (two-phase write step 1)

    Args:
        namespace (str):
        name (str):
        body (ArtifactInitiateRequest):  Example: {'annotations': {'git-commit': '8c1f4e2'},
            'description': 'ResNet-50 image-classification model pretrained on ImageNet.',
            'displayName': 'ResNet-50', 'labels': {'stage': 'production', 'team': 'vision'}, 'source':
            'webUpload', 'spec': {'format': 'safetensors', 'framework': 'pytorch', 'parameters':
            '25.6M', 'task': 'image-classification'}, 'version': '1.4.0', 'visibility': 'tenant'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ArtifactHubError | ArtifactInitiateResponse
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
    body: ArtifactInitiateRequest,
) -> Response[ArtifactHubError | ArtifactInitiateResponse]:
    """Initiate an artifact version (two-phase write step 1)

    Args:
        namespace (str):
        name (str):
        body (ArtifactInitiateRequest):  Example: {'annotations': {'git-commit': '8c1f4e2'},
            'description': 'ResNet-50 image-classification model pretrained on ImageNet.',
            'displayName': 'ResNet-50', 'labels': {'stage': 'production', 'team': 'vision'}, 'source':
            'webUpload', 'spec': {'format': 'safetensors', 'framework': 'pytorch', 'parameters':
            '25.6M', 'task': 'image-classification'}, 'version': '1.4.0', 'visibility': 'tenant'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ArtifactHubError | ArtifactInitiateResponse]
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
    body: ArtifactInitiateRequest,
) -> ArtifactHubError | ArtifactInitiateResponse | None:
    """Initiate an artifact version (two-phase write step 1)

    Args:
        namespace (str):
        name (str):
        body (ArtifactInitiateRequest):  Example: {'annotations': {'git-commit': '8c1f4e2'},
            'description': 'ResNet-50 image-classification model pretrained on ImageNet.',
            'displayName': 'ResNet-50', 'labels': {'stage': 'production', 'team': 'vision'}, 'source':
            'webUpload', 'spec': {'format': 'safetensors', 'framework': 'pytorch', 'parameters':
            '25.6M', 'task': 'image-classification'}, 'version': '1.4.0', 'visibility': 'tenant'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ArtifactHubError | ArtifactInitiateResponse
    """

    return (
        await asyncio_detailed(
            namespace=namespace,
            name=name,
            client=client,
            body=body,
        )
    ).parsed
