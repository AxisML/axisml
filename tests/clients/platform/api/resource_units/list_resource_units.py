from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.problem import Problem
from ...models.resource_unit_list import ResourceUnitList
from ...types import UNSET, Response, Unset


def _get_kwargs(
    pool: str,
    *,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
) -> dict[str, Any]:
    params: dict[str, Any] = {}

    params["limit"] = limit

    params["continue"] = continue_

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/resourcepools/{pool}/units".format(
            pool=quote(str(pool), safe=""),
        ),
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Problem | ResourceUnitList | None:
    if response.status_code == 200:
        response_200 = ResourceUnitList.from_dict(response.json())

        return response_200

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
) -> Response[Problem | ResourceUnitList]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    pool: str,
    *,
    client: AuthenticatedClient | Client,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
) -> Response[Problem | ResourceUnitList]:
    """List resource units in a pool

    Args:
        pool (str):
        limit (int | Unset):
        continue_ (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ResourceUnitList]
    """

    kwargs = _get_kwargs(
        pool=pool,
        limit=limit,
        continue_=continue_,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    pool: str,
    *,
    client: AuthenticatedClient | Client,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
) -> Problem | ResourceUnitList | None:
    """List resource units in a pool

    Args:
        pool (str):
        limit (int | Unset):
        continue_ (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ResourceUnitList
    """

    return sync_detailed(
        pool=pool,
        client=client,
        limit=limit,
        continue_=continue_,
    ).parsed


async def asyncio_detailed(
    pool: str,
    *,
    client: AuthenticatedClient | Client,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
) -> Response[Problem | ResourceUnitList]:
    """List resource units in a pool

    Args:
        pool (str):
        limit (int | Unset):
        continue_ (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | ResourceUnitList]
    """

    kwargs = _get_kwargs(
        pool=pool,
        limit=limit,
        continue_=continue_,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    pool: str,
    *,
    client: AuthenticatedClient | Client,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
) -> Problem | ResourceUnitList | None:
    """List resource units in a pool

    Args:
        pool (str):
        limit (int | Unset):
        continue_ (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | ResourceUnitList
    """

    return (
        await asyncio_detailed(
            pool=pool,
            client=client,
            limit=limit,
            continue_=continue_,
        )
    ).parsed
