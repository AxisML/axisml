from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.artifact import Artifact
from ...models.artifact_hub_error import ArtifactHubError
from ...types import Response


def _get_kwargs(
    namespace: str,
    name: str,
    version: str,
) -> dict[str, Any]:

    _kwargs: dict[str, Any] = {
        "method": "delete",
        "url": "/api/v1/namespaces/{namespace}/artifacts/{name}/{version}".format(
            namespace=quote(str(namespace), safe=""),
            name=quote(str(name), safe=""),
            version=quote(str(version), safe=""),
        ),
    }

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
) -> Response[Artifact | ArtifactHubError]:
    """Delete an artifact version

    Args:
        namespace (str):
        name (str):
        version (str):

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
) -> Artifact | ArtifactHubError | None:
    """Delete an artifact version

    Args:
        namespace (str):
        name (str):
        version (str):

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
    ).parsed


async def asyncio_detailed(
    namespace: str,
    name: str,
    version: str,
    *,
    client: AuthenticatedClient | Client,
) -> Response[Artifact | ArtifactHubError]:
    """Delete an artifact version

    Args:
        namespace (str):
        name (str):
        version (str):

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
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    namespace: str,
    name: str,
    version: str,
    *,
    client: AuthenticatedClient | Client,
) -> Artifact | ArtifactHubError | None:
    """Delete an artifact version

    Args:
        namespace (str):
        name (str):
        version (str):

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
        )
    ).parsed
