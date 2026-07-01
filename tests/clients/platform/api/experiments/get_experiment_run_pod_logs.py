from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.problem import Problem
from ...types import UNSET, Response, Unset


def _get_kwargs(
    name: str,
    run: str,
    pod: str,
    *,
    container: str | Unset = UNSET,
    tail_lines: int | Unset = UNSET,
    follow: bool | Unset = UNSET,
    previous: bool | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    if not isinstance(x_axisml_tenant, Unset):
        headers["X-Axisml-Tenant"] = x_axisml_tenant

    params: dict[str, Any] = {}

    params["container"] = container

    params["tailLines"] = tail_lines

    params["follow"] = follow

    params["previous"] = previous

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/experiments/{name}/runs/{run}/pods/{pod}/logs".format(
            name=quote(str(name), safe=""),
            run=quote(str(run), safe=""),
            pod=quote(str(pod), safe=""),
        ),
        "params": params,
    }

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Problem | str | None:
    if response.status_code == 200:
        response_200 = response.text
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

    if response.status_code == 410:
        response_410 = Problem.from_dict(response.json())

        return response_410

    if response.status_code == 500:
        response_500 = Problem.from_dict(response.json())

        return response_500

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Problem | str]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    name: str,
    run: str,
    pod: str,
    *,
    client: AuthenticatedClient | Client,
    container: str | Unset = UNSET,
    tail_lines: int | Unset = UNSET,
    follow: bool | Unset = UNSET,
    previous: bool | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[Problem | str]:
    """Fetch or stream a Run pod's logs

    Args:
        name (str):
        run (str):
        pod (str):
        container (str | Unset):
        tail_lines (int | Unset):
        follow (bool | Unset):
        previous (bool | Unset):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | str]
    """

    kwargs = _get_kwargs(
        name=name,
        run=run,
        pod=pod,
        container=container,
        tail_lines=tail_lines,
        follow=follow,
        previous=previous,
        x_axisml_tenant=x_axisml_tenant,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    name: str,
    run: str,
    pod: str,
    *,
    client: AuthenticatedClient | Client,
    container: str | Unset = UNSET,
    tail_lines: int | Unset = UNSET,
    follow: bool | Unset = UNSET,
    previous: bool | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> Problem | str | None:
    """Fetch or stream a Run pod's logs

    Args:
        name (str):
        run (str):
        pod (str):
        container (str | Unset):
        tail_lines (int | Unset):
        follow (bool | Unset):
        previous (bool | Unset):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | str
    """

    return sync_detailed(
        name=name,
        run=run,
        pod=pod,
        client=client,
        container=container,
        tail_lines=tail_lines,
        follow=follow,
        previous=previous,
        x_axisml_tenant=x_axisml_tenant,
    ).parsed


async def asyncio_detailed(
    name: str,
    run: str,
    pod: str,
    *,
    client: AuthenticatedClient | Client,
    container: str | Unset = UNSET,
    tail_lines: int | Unset = UNSET,
    follow: bool | Unset = UNSET,
    previous: bool | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[Problem | str]:
    """Fetch or stream a Run pod's logs

    Args:
        name (str):
        run (str):
        pod (str):
        container (str | Unset):
        tail_lines (int | Unset):
        follow (bool | Unset):
        previous (bool | Unset):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | str]
    """

    kwargs = _get_kwargs(
        name=name,
        run=run,
        pod=pod,
        container=container,
        tail_lines=tail_lines,
        follow=follow,
        previous=previous,
        x_axisml_tenant=x_axisml_tenant,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    name: str,
    run: str,
    pod: str,
    *,
    client: AuthenticatedClient | Client,
    container: str | Unset = UNSET,
    tail_lines: int | Unset = UNSET,
    follow: bool | Unset = UNSET,
    previous: bool | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> Problem | str | None:
    """Fetch or stream a Run pod's logs

    Args:
        name (str):
        run (str):
        pod (str):
        container (str | Unset):
        tail_lines (int | Unset):
        follow (bool | Unset):
        previous (bool | Unset):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | str
    """

    return (
        await asyncio_detailed(
            name=name,
            run=run,
            pod=pod,
            client=client,
            container=container,
            tail_lines=tail_lines,
            follow=follow,
            previous=previous,
            x_axisml_tenant=x_axisml_tenant,
        )
    ).parsed
