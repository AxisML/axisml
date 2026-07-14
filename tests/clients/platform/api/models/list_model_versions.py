from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.model_list import ModelList
from ...models.model_status import ModelStatus
from ...models.problem import Problem
from ...types import UNSET, Response, Unset


def _get_kwargs(
    tenant: str,
    name: str,
    *,
    status: ModelStatus | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
) -> dict[str, Any]:
    params: dict[str, Any] = {}

    json_status: str | Unset = UNSET
    if not isinstance(status, Unset):
        json_status = status.value

    params["status"] = json_status

    params["limit"] = limit

    params["continue"] = continue_

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/models/{tenant}/{name}/versions".format(
            tenant=quote(str(tenant), safe=""),
            name=quote(str(name), safe=""),
        ),
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ModelList | Problem | None:
    if response.status_code == 200:
        response_200 = ModelList.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Problem.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = Problem.from_dict(response.json())

        return response_401

    if response.status_code == 404:
        response_404 = Problem.from_dict(response.json())

        return response_404

    if response.status_code == 500:
        response_500 = Problem.from_dict(response.json())

        return response_500

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ModelList | Problem]:
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
    status: ModelStatus | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
) -> Response[ModelList | Problem]:
    """List versions of a model definition (live)

    Args:
        tenant (str):
        name (str):
        status (ModelStatus | Unset): Mirrors artifacts ArtifactStatus for kind=model.
        limit (int | Unset):
        continue_ (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ModelList | Problem]
    """

    kwargs = _get_kwargs(
        tenant=tenant,
        name=name,
        status=status,
        limit=limit,
        continue_=continue_,
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
    status: ModelStatus | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
) -> ModelList | Problem | None:
    """List versions of a model definition (live)

    Args:
        tenant (str):
        name (str):
        status (ModelStatus | Unset): Mirrors artifacts ArtifactStatus for kind=model.
        limit (int | Unset):
        continue_ (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ModelList | Problem
    """

    return sync_detailed(
        tenant=tenant,
        name=name,
        client=client,
        status=status,
        limit=limit,
        continue_=continue_,
    ).parsed


async def asyncio_detailed(
    tenant: str,
    name: str,
    *,
    client: AuthenticatedClient | Client,
    status: ModelStatus | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
) -> Response[ModelList | Problem]:
    """List versions of a model definition (live)

    Args:
        tenant (str):
        name (str):
        status (ModelStatus | Unset): Mirrors artifacts ArtifactStatus for kind=model.
        limit (int | Unset):
        continue_ (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ModelList | Problem]
    """

    kwargs = _get_kwargs(
        tenant=tenant,
        name=name,
        status=status,
        limit=limit,
        continue_=continue_,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    tenant: str,
    name: str,
    *,
    client: AuthenticatedClient | Client,
    status: ModelStatus | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
) -> ModelList | Problem | None:
    """List versions of a model definition (live)

    Args:
        tenant (str):
        name (str):
        status (ModelStatus | Unset): Mirrors artifacts ArtifactStatus for kind=model.
        limit (int | Unset):
        continue_ (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ModelList | Problem
    """

    return (
        await asyncio_detailed(
            tenant=tenant,
            name=name,
            client=client,
            status=status,
            limit=limit,
            continue_=continue_,
        )
    ).parsed
