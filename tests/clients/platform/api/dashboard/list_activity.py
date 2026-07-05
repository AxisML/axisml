from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.activity_list import ActivityList
from ...models.problem import Problem
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    limit: int | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    if not isinstance(x_axisml_tenant, Unset):
        headers["X-Axisml-Tenant"] = x_axisml_tenant

    params: dict[str, Any] = {}

    params["limit"] = limit

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/dashboard/activity",
        "params": params,
    }

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ActivityList | Problem | None:
    if response.status_code == 200:
        response_200 = ActivityList.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Problem.from_dict(response.json())

        return response_400

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
) -> Response[ActivityList | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    limit: int | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[ActivityList | Problem]:
    """Recent-activity feed for the active tenant

    Args:
        limit (int | Unset):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ActivityList | Problem]
    """

    kwargs = _get_kwargs(
        limit=limit,
        x_axisml_tenant=x_axisml_tenant,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    limit: int | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> ActivityList | Problem | None:
    """Recent-activity feed for the active tenant

    Args:
        limit (int | Unset):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ActivityList | Problem
    """

    return sync_detailed(
        client=client,
        limit=limit,
        x_axisml_tenant=x_axisml_tenant,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    limit: int | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[ActivityList | Problem]:
    """Recent-activity feed for the active tenant

    Args:
        limit (int | Unset):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ActivityList | Problem]
    """

    kwargs = _get_kwargs(
        limit=limit,
        x_axisml_tenant=x_axisml_tenant,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    limit: int | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> ActivityList | Problem | None:
    """Recent-activity feed for the active tenant

    Args:
        limit (int | Unset):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ActivityList | Problem
    """

    return (
        await asyncio_detailed(
            client=client,
            limit=limit,
            x_axisml_tenant=x_axisml_tenant,
        )
    ).parsed
