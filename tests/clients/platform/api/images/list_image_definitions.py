from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.artifact_definition_list import ArtifactDefinitionList
from ...models.image_purpose import ImagePurpose
from ...models.problem import Problem
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    q: str | Unset = UNSET,
    purpose: ImagePurpose | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    if not isinstance(x_axisml_tenant, Unset):
        headers["X-Axisml-Tenant"] = x_axisml_tenant

    params: dict[str, Any] = {}

    params["q"] = q

    json_purpose: str | Unset = UNSET
    if not isinstance(purpose, Unset):
        json_purpose = purpose.value

    params["purpose"] = json_purpose

    params["limit"] = limit

    params["continue"] = continue_

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/images",
        "params": params,
    }

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ArtifactDefinitionList | Problem | None:
    if response.status_code == 200:
        response_200 = ArtifactDefinitionList.from_dict(response.json())

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
) -> Response[ArtifactDefinitionList | Problem]:
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
    purpose: ImagePurpose | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[ArtifactDefinitionList | Problem]:
    """List image definitions

    Args:
        q (str | Unset):
        purpose (ImagePurpose | Unset): Image definition intended use (list filter).
        limit (int | Unset):
        continue_ (str | Unset):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ArtifactDefinitionList | Problem]
    """

    kwargs = _get_kwargs(
        q=q,
        purpose=purpose,
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
    q: str | Unset = UNSET,
    purpose: ImagePurpose | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> ArtifactDefinitionList | Problem | None:
    """List image definitions

    Args:
        q (str | Unset):
        purpose (ImagePurpose | Unset): Image definition intended use (list filter).
        limit (int | Unset):
        continue_ (str | Unset):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ArtifactDefinitionList | Problem
    """

    return sync_detailed(
        client=client,
        q=q,
        purpose=purpose,
        limit=limit,
        continue_=continue_,
        x_axisml_tenant=x_axisml_tenant,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    q: str | Unset = UNSET,
    purpose: ImagePurpose | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[ArtifactDefinitionList | Problem]:
    """List image definitions

    Args:
        q (str | Unset):
        purpose (ImagePurpose | Unset): Image definition intended use (list filter).
        limit (int | Unset):
        continue_ (str | Unset):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ArtifactDefinitionList | Problem]
    """

    kwargs = _get_kwargs(
        q=q,
        purpose=purpose,
        limit=limit,
        continue_=continue_,
        x_axisml_tenant=x_axisml_tenant,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    q: str | Unset = UNSET,
    purpose: ImagePurpose | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    x_axisml_tenant: str | Unset = UNSET,
) -> ArtifactDefinitionList | Problem | None:
    """List image definitions

    Args:
        q (str | Unset):
        purpose (ImagePurpose | Unset): Image definition intended use (list filter).
        limit (int | Unset):
        continue_ (str | Unset):
        x_axisml_tenant (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ArtifactDefinitionList | Problem
    """

    return (
        await asyncio_detailed(
            client=client,
            q=q,
            purpose=purpose,
            limit=limit,
            continue_=continue_,
            x_axisml_tenant=x_axisml_tenant,
        )
    ).parsed
