from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.problem import Problem
from ...models.run import Run
from ...types import UNSET, Response, Unset


def _get_kwargs(
    name: str,
    run: str,
    *,
    x_axisml_tenant: str | Unset = UNSET,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    if not isinstance(x_axisml_tenant, Unset):
        headers["X-Axisml-Tenant"] = x_axisml_tenant

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/experiments/{name}/runs/{run}/cancel".format(
            name=quote(str(name), safe=""),
            run=quote(str(run), safe=""),
        ),
    }

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Problem | Run | None:
    if response.status_code == 200:
        response_200 = Run.from_dict(response.json())

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

    if response.status_code == 404:
        response_404 = Problem.from_dict(response.json())

        return response_404

    if response.status_code == 409:
        response_409 = Problem.from_dict(response.json())

        return response_409

    if response.status_code == 500:
        response_500 = Problem.from_dict(response.json())

        return response_500

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Problem | Run]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    name: str,
    run: str,
    *,
    client: AuthenticatedClient | Client,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[Problem | Run]:
    """Cancel a running Run

    Args:
        name (str):
        run (str):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | Run]
    """

    kwargs = _get_kwargs(
        name=name,
        run=run,
        x_axisml_tenant=x_axisml_tenant,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    name: str,
    run: str,
    *,
    client: AuthenticatedClient | Client,
    x_axisml_tenant: str | Unset = UNSET,
) -> Problem | Run | None:
    """Cancel a running Run

    Args:
        name (str):
        run (str):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | Run
    """

    return sync_detailed(
        name=name,
        run=run,
        client=client,
        x_axisml_tenant=x_axisml_tenant,
    ).parsed


async def asyncio_detailed(
    name: str,
    run: str,
    *,
    client: AuthenticatedClient | Client,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[Problem | Run]:
    """Cancel a running Run

    Args:
        name (str):
        run (str):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | Run]
    """

    kwargs = _get_kwargs(
        name=name,
        run=run,
        x_axisml_tenant=x_axisml_tenant,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    name: str,
    run: str,
    *,
    client: AuthenticatedClient | Client,
    x_axisml_tenant: str | Unset = UNSET,
) -> Problem | Run | None:
    """Cancel a running Run

    Args:
        name (str):
        run (str):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | Run
    """

    return (
        await asyncio_detailed(
            name=name,
            run=run,
            client=client,
            x_axisml_tenant=x_axisml_tenant,
        )
    ).parsed
