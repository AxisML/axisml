from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.job_list import JobList
from ...models.problem import Problem
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    owner: str | Unset = UNSET,
    q: str | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    if not isinstance(x_axisml_tenant, Unset):
        headers["X-Axisml-Tenant"] = x_axisml_tenant

    params: dict[str, Any] = {}

    params["owner"] = owner

    params["q"] = q

    params["limit"] = limit

    params["continue"] = continue_

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/jobs",
        "params": params,
    }

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> JobList | Problem | None:
    if response.status_code == 200:
        response_200 = JobList.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Problem.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = Problem.from_dict(response.json())

        return response_401

    if response.status_code == 403:
        response_403 = Problem.from_dict(response.json())

        return response_403

    if response.status_code == 500:
        response_500 = Problem.from_dict(response.json())

        return response_500

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[JobList | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    owner: str | Unset = UNSET,
    q: str | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[JobList | Problem]:
    """List Jobs

    Args:
        owner (str | Unset):
        q (str | Unset):
        limit (int | Unset):
        continue_ (str | Unset):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[JobList | Problem]
    """

    kwargs = _get_kwargs(
        owner=owner,
        q=q,
        limit=limit,
        continue_=continue_,
        x_axisml_tenant=x_axisml_tenant,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    owner: str | Unset = UNSET,
    q: str | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> JobList | Problem | None:
    """List Jobs

    Args:
        owner (str | Unset):
        q (str | Unset):
        limit (int | Unset):
        continue_ (str | Unset):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        JobList | Problem
    """

    return sync_detailed(
        client=client,
        owner=owner,
        q=q,
        limit=limit,
        continue_=continue_,
        x_axisml_tenant=x_axisml_tenant,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    owner: str | Unset = UNSET,
    q: str | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[JobList | Problem]:
    """List Jobs

    Args:
        owner (str | Unset):
        q (str | Unset):
        limit (int | Unset):
        continue_ (str | Unset):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[JobList | Problem]
    """

    kwargs = _get_kwargs(
        owner=owner,
        q=q,
        limit=limit,
        continue_=continue_,
        x_axisml_tenant=x_axisml_tenant,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    owner: str | Unset = UNSET,
    q: str | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> JobList | Problem | None:
    """List Jobs

    Args:
        owner (str | Unset):
        q (str | Unset):
        limit (int | Unset):
        continue_ (str | Unset):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        JobList | Problem
    """

    return (
        await asyncio_detailed(
            client=client,
            owner=owner,
            q=q,
            limit=limit,
            continue_=continue_,
            x_axisml_tenant=x_axisml_tenant,
        )
    ).parsed
