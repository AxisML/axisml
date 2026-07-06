from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.artifact_hub_error import ArtifactHubError
from ...models.artifact_list import ArtifactList
from ...types import UNSET, Response, Unset


def _get_kwargs(
    namespace: str,
    name: str,
    *,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    status: str | Unset = UNSET,
    label_selector: str | Unset = UNSET,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["limit"] = limit

    params["continue"] = continue_

    params["status"] = status

    params["labelSelector"] = label_selector

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/namespaces/{namespace}/artifacts/{name}".format(
            namespace=quote(str(namespace), safe=""),
            name=quote(str(name), safe=""),
        ),
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ArtifactHubError | ArtifactList:
    if response.status_code == 200:
        response_200 = ArtifactList.from_dict(response.json())

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
) -> Response[ArtifactHubError | ArtifactList]:
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
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    status: str | Unset = UNSET,
    label_selector: str | Unset = UNSET,
) -> Response[ArtifactHubError | ArtifactList]:
    """List versions of an artifact

    Args:
        namespace (str):
        name (str):
        limit (int | Unset):
        continue_ (str | Unset):
        status (str | Unset):
        label_selector (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ArtifactHubError | ArtifactList]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        name=name,
        limit=limit,
        continue_=continue_,
        status=status,
        label_selector=label_selector,
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
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    status: str | Unset = UNSET,
    label_selector: str | Unset = UNSET,
) -> ArtifactHubError | ArtifactList | None:
    """List versions of an artifact

    Args:
        namespace (str):
        name (str):
        limit (int | Unset):
        continue_ (str | Unset):
        status (str | Unset):
        label_selector (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ArtifactHubError | ArtifactList
    """

    return sync_detailed(
        namespace=namespace,
        name=name,
        client=client,
        limit=limit,
        continue_=continue_,
        status=status,
        label_selector=label_selector,
    ).parsed


async def asyncio_detailed(
    namespace: str,
    name: str,
    *,
    client: AuthenticatedClient | Client,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    status: str | Unset = UNSET,
    label_selector: str | Unset = UNSET,
) -> Response[ArtifactHubError | ArtifactList]:
    """List versions of an artifact

    Args:
        namespace (str):
        name (str):
        limit (int | Unset):
        continue_ (str | Unset):
        status (str | Unset):
        label_selector (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ArtifactHubError | ArtifactList]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        name=name,
        limit=limit,
        continue_=continue_,
        status=status,
        label_selector=label_selector,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    namespace: str,
    name: str,
    *,
    client: AuthenticatedClient | Client,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    status: str | Unset = UNSET,
    label_selector: str | Unset = UNSET,
) -> ArtifactHubError | ArtifactList | None:
    """List versions of an artifact

    Args:
        namespace (str):
        name (str):
        limit (int | Unset):
        continue_ (str | Unset):
        status (str | Unset):
        label_selector (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ArtifactHubError | ArtifactList
    """

    return (
        await asyncio_detailed(
            namespace=namespace,
            name=name,
            client=client,
            limit=limit,
            continue_=continue_,
            status=status,
            label_selector=label_selector,
        )
    ).parsed
