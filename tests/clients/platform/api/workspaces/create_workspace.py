from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.problem import Problem
from ...models.workspace import Workspace
from ...models.workspace_create_request import WorkspaceCreateRequest
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    body: WorkspaceCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    if not isinstance(x_axisml_tenant, Unset):
        headers["X-Axisml-Tenant"] = x_axisml_tenant

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/workspaces",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Problem | Workspace | None:
    if response.status_code == 201:
        response_201 = Workspace.from_dict(response.json())

        return response_201

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

    if response.status_code == 422:
        response_422 = Problem.from_dict(response.json())

        return response_422

    if response.status_code == 500:
        response_500 = Problem.from_dict(response.json())

        return response_500

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Problem | Workspace]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: WorkspaceCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[Problem | Workspace]:
    """Create a workspace

    Args:
        x_axisml_tenant (str | Unset):
        body (WorkspaceCreateRequest):  Example: {'containerPort': 8888, 'description':
            'JupyterLab interactive development environment.', 'displayName': 'Vision team dev
            environment', 'image': 'registry.axisml.io/dev/jupyter:3.0.0', 'lifecycle':
            {'idleTimeoutSeconds': 3600}, 'name': 'notebook-dev', 'poolName': 'gpu-a100', 'unitName':
            'a100-1x', 'volumes': [{'mountPath': '/home/jovyan/work', 'name': 'notebook-data'}]}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | Workspace]
    """

    kwargs = _get_kwargs(
        body=body,
        x_axisml_tenant=x_axisml_tenant,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    body: WorkspaceCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> Problem | Workspace | None:
    """Create a workspace

    Args:
        x_axisml_tenant (str | Unset):
        body (WorkspaceCreateRequest):  Example: {'containerPort': 8888, 'description':
            'JupyterLab interactive development environment.', 'displayName': 'Vision team dev
            environment', 'image': 'registry.axisml.io/dev/jupyter:3.0.0', 'lifecycle':
            {'idleTimeoutSeconds': 3600}, 'name': 'notebook-dev', 'poolName': 'gpu-a100', 'unitName':
            'a100-1x', 'volumes': [{'mountPath': '/home/jovyan/work', 'name': 'notebook-data'}]}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | Workspace
    """

    return sync_detailed(
        client=client,
        body=body,
        x_axisml_tenant=x_axisml_tenant,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: WorkspaceCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[Problem | Workspace]:
    """Create a workspace

    Args:
        x_axisml_tenant (str | Unset):
        body (WorkspaceCreateRequest):  Example: {'containerPort': 8888, 'description':
            'JupyterLab interactive development environment.', 'displayName': 'Vision team dev
            environment', 'image': 'registry.axisml.io/dev/jupyter:3.0.0', 'lifecycle':
            {'idleTimeoutSeconds': 3600}, 'name': 'notebook-dev', 'poolName': 'gpu-a100', 'unitName':
            'a100-1x', 'volumes': [{'mountPath': '/home/jovyan/work', 'name': 'notebook-data'}]}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | Workspace]
    """

    kwargs = _get_kwargs(
        body=body,
        x_axisml_tenant=x_axisml_tenant,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: WorkspaceCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> Problem | Workspace | None:
    """Create a workspace

    Args:
        x_axisml_tenant (str | Unset):
        body (WorkspaceCreateRequest):  Example: {'containerPort': 8888, 'description':
            'JupyterLab interactive development environment.', 'displayName': 'Vision team dev
            environment', 'image': 'registry.axisml.io/dev/jupyter:3.0.0', 'lifecycle':
            {'idleTimeoutSeconds': 3600}, 'name': 'notebook-dev', 'poolName': 'gpu-a100', 'unitName':
            'a100-1x', 'volumes': [{'mountPath': '/home/jovyan/work', 'name': 'notebook-data'}]}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | Workspace
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
            x_axisml_tenant=x_axisml_tenant,
        )
    ).parsed
