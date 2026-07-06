from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.artifact import Artifact
from ...models.artifact_complete_request import ArtifactCompleteRequest
from ...models.artifact_hub_error import ArtifactHubError
from ...types import Response


def _get_kwargs(
    namespace: str,
    name: str,
    version: str,
    *,
    body: ArtifactCompleteRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/namespaces/{namespace}/artifacts/{name}/{version}/complete".format(
            namespace=quote(str(namespace), safe=""),
            name=quote(str(name), safe=""),
            version=quote(str(version), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Artifact | ArtifactHubError:
    if response.status_code == 200:
        response_200 = Artifact.from_dict(response.json())

        return response_200

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
) -> Response[Artifact | ArtifactHubError]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    namespace: str,
    name: str,
    version: str,
    *,
    client: AuthenticatedClient | Client,
    body: ArtifactCompleteRequest,
) -> Response[Artifact | ArtifactHubError]:
    """Complete artifact upload (two-phase write step 2)

    Args:
        namespace (str):
        name (str):
        version (str):
        body (ArtifactCompleteRequest):  Example: {'digest':
            'sha256:9b2c1f4e22a74c0e9b1d7f3a2e5c9a108c1f4e222b7a4c0e9b1d7f3a2e5c9a10'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Artifact | ArtifactHubError]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        name=name,
        version=version,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    namespace: str,
    name: str,
    version: str,
    *,
    client: AuthenticatedClient | Client,
    body: ArtifactCompleteRequest,
) -> Artifact | ArtifactHubError | None:
    """Complete artifact upload (two-phase write step 2)

    Args:
        namespace (str):
        name (str):
        version (str):
        body (ArtifactCompleteRequest):  Example: {'digest':
            'sha256:9b2c1f4e22a74c0e9b1d7f3a2e5c9a108c1f4e222b7a4c0e9b1d7f3a2e5c9a10'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Artifact | ArtifactHubError
    """

    return sync_detailed(
        namespace=namespace,
        name=name,
        version=version,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    namespace: str,
    name: str,
    version: str,
    *,
    client: AuthenticatedClient | Client,
    body: ArtifactCompleteRequest,
) -> Response[Artifact | ArtifactHubError]:
    """Complete artifact upload (two-phase write step 2)

    Args:
        namespace (str):
        name (str):
        version (str):
        body (ArtifactCompleteRequest):  Example: {'digest':
            'sha256:9b2c1f4e22a74c0e9b1d7f3a2e5c9a108c1f4e222b7a4c0e9b1d7f3a2e5c9a10'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Artifact | ArtifactHubError]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        name=name,
        version=version,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    namespace: str,
    name: str,
    version: str,
    *,
    client: AuthenticatedClient | Client,
    body: ArtifactCompleteRequest,
) -> Artifact | ArtifactHubError | None:
    """Complete artifact upload (two-phase write step 2)

    Args:
        namespace (str):
        name (str):
        version (str):
        body (ArtifactCompleteRequest):  Example: {'digest':
            'sha256:9b2c1f4e22a74c0e9b1d7f3a2e5c9a108c1f4e222b7a4c0e9b1d7f3a2e5c9a10'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Artifact | ArtifactHubError
    """

    return (
        await asyncio_detailed(
            namespace=namespace,
            name=name,
            version=version,
            client=client,
            body=body,
        )
    ).parsed
