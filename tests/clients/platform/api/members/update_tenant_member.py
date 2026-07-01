from http import HTTPStatus
from typing import Any
from urllib.parse import quote
from uuid import UUID

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.member import Member
from ...models.member_patch_request import MemberPatchRequest
from ...models.problem import Problem
from ...types import Response


def _get_kwargs(
    name: str,
    user_id: UUID,
    *,
    body: MemberPatchRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "patch",
        "url": "/api/v1/tenants/{name}/members/{user_id}".format(
            name=quote(str(name), safe=""),
            user_id=quote(str(user_id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Member | Problem | None:
    if response.status_code == 200:
        response_200 = Member.from_dict(response.json())

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
) -> Response[Member | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    name: str,
    user_id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: MemberPatchRequest,
) -> Response[Member | Problem]:
    """Change a member's role

    Args:
        name (str):
        user_id (UUID):
        body (MemberPatchRequest):  Example: {'roleName': 'tenant-admin'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Member | Problem]
    """

    kwargs = _get_kwargs(
        name=name,
        user_id=user_id,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    name: str,
    user_id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: MemberPatchRequest,
) -> Member | Problem | None:
    """Change a member's role

    Args:
        name (str):
        user_id (UUID):
        body (MemberPatchRequest):  Example: {'roleName': 'tenant-admin'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Member | Problem
    """

    return sync_detailed(
        name=name,
        user_id=user_id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    name: str,
    user_id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: MemberPatchRequest,
) -> Response[Member | Problem]:
    """Change a member's role

    Args:
        name (str):
        user_id (UUID):
        body (MemberPatchRequest):  Example: {'roleName': 'tenant-admin'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Member | Problem]
    """

    kwargs = _get_kwargs(
        name=name,
        user_id=user_id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    name: str,
    user_id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: MemberPatchRequest,
) -> Member | Problem | None:
    """Change a member's role

    Args:
        name (str):
        user_id (UUID):
        body (MemberPatchRequest):  Example: {'roleName': 'tenant-admin'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Member | Problem
    """

    return (
        await asyncio_detailed(
            name=name,
            user_id=user_id,
            client=client,
            body=body,
        )
    ).parsed
