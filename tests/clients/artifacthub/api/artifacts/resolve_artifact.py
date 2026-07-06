from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.artifact_hub_error import ArtifactHubError
from ...models.artifact_resolve_response import ArtifactResolveResponse
from ...types import UNSET, Response, Unset


def _get_kwargs(
    namespace: str,
    name: str,
    version: str,
    *,
    usage: str | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["usage"] = usage

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/namespaces/{namespace}/artifacts/{name}/{version}/resolve".format(
            namespace=quote(str(namespace), safe=""),
            name=quote(str(name), safe=""),
            version=quote(str(version), safe=""),
        ),
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ArtifactHubError | ArtifactResolveResponse:
    if response.status_code == 200:
        response_200 = ArtifactResolveResponse.from_dict(response.json())

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
) -> Response[ArtifactHubError | ArtifactResolveResponse]:
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
    usage: str | Unset = UNSET,
) -> Response[ArtifactHubError | ArtifactResolveResponse]:
    """Resolve an artifact for download

    Args:
        namespace (str):
        name (str):
        version (str):
        usage (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ArtifactHubError | ArtifactResolveResponse]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        name=name,
        version=version,
        usage=usage,
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
    usage: str | Unset = UNSET,
) -> ArtifactHubError | ArtifactResolveResponse | None:
    """Resolve an artifact for download

    Args:
        namespace (str):
        name (str):
        version (str):
        usage (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ArtifactHubError | ArtifactResolveResponse
    """

    return sync_detailed(
        namespace=namespace,
        name=name,
        version=version,
        client=client,
        usage=usage,
    ).parsed


async def asyncio_detailed(
    namespace: str,
    name: str,
    version: str,
    *,
    client: AuthenticatedClient | Client,
    usage: str | Unset = UNSET,
) -> Response[ArtifactHubError | ArtifactResolveResponse]:
    """Resolve an artifact for download

    Args:
        namespace (str):
        name (str):
        version (str):
        usage (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ArtifactHubError | ArtifactResolveResponse]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        name=name,
        version=version,
        usage=usage,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    namespace: str,
    name: str,
    version: str,
    *,
    client: AuthenticatedClient | Client,
    usage: str | Unset = UNSET,
) -> ArtifactHubError | ArtifactResolveResponse | None:
    """Resolve an artifact for download

    Args:
        namespace (str):
        name (str):
        version (str):
        usage (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ArtifactHubError | ArtifactResolveResponse
    """

    return (
        await asyncio_detailed(
            namespace=namespace,
            name=name,
            version=version,
            client=client,
            usage=usage,
        )
    ).parsed
