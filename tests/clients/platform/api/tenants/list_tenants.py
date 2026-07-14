from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.problem import Problem
from ...models.tenant_list import TenantList
from ...models.tenant_phase import TenantPhase
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    q: str | Unset = UNSET,
    phase: TenantPhase | Unset = UNSET,
    stats: bool | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
) -> dict[str, Any]:
    params: dict[str, Any] = {}

    params["q"] = q

    json_phase: str | Unset = UNSET
    if not isinstance(phase, Unset):
        json_phase = phase.value

    params["phase"] = json_phase

    params["stats"] = stats

    params["limit"] = limit

    params["continue"] = continue_

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/tenants",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Problem | TenantList | None:
    if response.status_code == 200:
        response_200 = TenantList.from_dict(response.json())

        return response_200

    if response.status_code == 401:
        response_401 = Problem.from_dict(response.json())

        return response_401

    if response.status_code == 500:
        response_500 = Problem.from_dict(response.json())

        return response_500

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Problem | TenantList]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    q: str | Unset = UNSET,
    phase: TenantPhase | Unset = UNSET,
    stats: bool | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
) -> Response[Problem | TenantList]:
    """List visible tenants

    Args:
        q (str | Unset):
        phase (TenantPhase | Unset):
        stats (bool | Unset):
        limit (int | Unset):
        continue_ (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TenantList]
    """

    kwargs = _get_kwargs(
        q=q,
        phase=phase,
        stats=stats,
        limit=limit,
        continue_=continue_,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    q: str | Unset = UNSET,
    phase: TenantPhase | Unset = UNSET,
    stats: bool | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
) -> Problem | TenantList | None:
    """List visible tenants

    Args:
        q (str | Unset):
        phase (TenantPhase | Unset):
        stats (bool | Unset):
        limit (int | Unset):
        continue_ (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TenantList
    """

    return sync_detailed(
        client=client,
        q=q,
        phase=phase,
        stats=stats,
        limit=limit,
        continue_=continue_,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    q: str | Unset = UNSET,
    phase: TenantPhase | Unset = UNSET,
    stats: bool | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
) -> Response[Problem | TenantList]:
    """List visible tenants

    Args:
        q (str | Unset):
        phase (TenantPhase | Unset):
        stats (bool | Unset):
        limit (int | Unset):
        continue_ (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TenantList]
    """

    kwargs = _get_kwargs(
        q=q,
        phase=phase,
        stats=stats,
        limit=limit,
        continue_=continue_,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    q: str | Unset = UNSET,
    phase: TenantPhase | Unset = UNSET,
    stats: bool | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
) -> Problem | TenantList | None:
    """List visible tenants

    Args:
        q (str | Unset):
        phase (TenantPhase | Unset):
        stats (bool | Unset):
        limit (int | Unset):
        continue_ (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TenantList
    """

    return (
        await asyncio_detailed(
            client=client,
            q=q,
            phase=phase,
            stats=stats,
            limit=limit,
            continue_=continue_,
        )
    ).parsed
