from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.problem import Problem
from ...models.run_list import RunList
from ...models.run_phase import RunPhase
from ...types import UNSET, Response, Unset


def _get_kwargs(
    name: str,
    *,
    phase: RunPhase | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    if not isinstance(x_axisml_tenant, Unset):
        headers["X-Axisml-Tenant"] = x_axisml_tenant

    params: dict[str, Any] = {}

    json_phase: str | Unset = UNSET
    if not isinstance(phase, Unset):
        json_phase = phase.value

    params["phase"] = json_phase

    params["limit"] = limit

    params["continue"] = continue_

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/experiments/{name}/runs".format(
            name=quote(str(name), safe=""),
        ),
        "params": params,
    }

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Problem | RunList | None:
    if response.status_code == 200:
        response_200 = RunList.from_dict(response.json())

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

    if response.status_code == 500:
        response_500 = Problem.from_dict(response.json())

        return response_500

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Problem | RunList]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    name: str,
    *,
    client: AuthenticatedClient | Client,
    phase: RunPhase | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[Problem | RunList]:
    """List Runs of an experiment (live)

    Args:
        name (str):
        phase (RunPhase | Unset): Run (compute MLRun) phase. The active (non-terminal) phases —
            Creating / Pending / Running / Canceling — block Job-definition deletion.
        limit (int | Unset):
        continue_ (str | Unset):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | RunList]
    """

    kwargs = _get_kwargs(
        name=name,
        phase=phase,
        limit=limit,
        continue_=continue_,
        x_axisml_tenant=x_axisml_tenant,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    name: str,
    *,
    client: AuthenticatedClient | Client,
    phase: RunPhase | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> Problem | RunList | None:
    """List Runs of an experiment (live)

    Args:
        name (str):
        phase (RunPhase | Unset): Run (compute MLRun) phase. The active (non-terminal) phases —
            Creating / Pending / Running / Canceling — block Job-definition deletion.
        limit (int | Unset):
        continue_ (str | Unset):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | RunList
    """

    return sync_detailed(
        name=name,
        client=client,
        phase=phase,
        limit=limit,
        continue_=continue_,
        x_axisml_tenant=x_axisml_tenant,
    ).parsed


async def asyncio_detailed(
    name: str,
    *,
    client: AuthenticatedClient | Client,
    phase: RunPhase | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[Problem | RunList]:
    """List Runs of an experiment (live)

    Args:
        name (str):
        phase (RunPhase | Unset): Run (compute MLRun) phase. The active (non-terminal) phases —
            Creating / Pending / Running / Canceling — block Job-definition deletion.
        limit (int | Unset):
        continue_ (str | Unset):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | RunList]
    """

    kwargs = _get_kwargs(
        name=name,
        phase=phase,
        limit=limit,
        continue_=continue_,
        x_axisml_tenant=x_axisml_tenant,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    name: str,
    *,
    client: AuthenticatedClient | Client,
    phase: RunPhase | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> Problem | RunList | None:
    """List Runs of an experiment (live)

    Args:
        name (str):
        phase (RunPhase | Unset): Run (compute MLRun) phase. The active (non-terminal) phases —
            Creating / Pending / Running / Canceling — block Job-definition deletion.
        limit (int | Unset):
        continue_ (str | Unset):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | RunList
    """

    return (
        await asyncio_detailed(
            name=name,
            client=client,
            phase=phase,
            limit=limit,
            continue_=continue_,
            x_axisml_tenant=x_axisml_tenant,
        )
    ).parsed
