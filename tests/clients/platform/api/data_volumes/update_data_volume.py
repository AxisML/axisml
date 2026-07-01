from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.data_volume import DataVolume
from ...models.data_volume_patch_request import DataVolumePatchRequest
from ...models.problem import Problem
from ...types import UNSET, Response, Unset


def _get_kwargs(
    name: str,
    *,
    body: DataVolumePatchRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    if not isinstance(x_axisml_tenant, Unset):
        headers["X-Axisml-Tenant"] = x_axisml_tenant

    _kwargs: dict[str, Any] = {
        "method": "patch",
        "url": "/api/v1/datavolumes/{name}".format(
            name=quote(str(name), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> DataVolume | Problem | None:
    if response.status_code == 200:
        response_200 = DataVolume.from_dict(response.json())

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
) -> Response[DataVolume | Problem]:
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
    body: DataVolumePatchRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[DataVolume | Problem]:
    """Expand or relabel a data volume

    Args:
        name (str):
        x_axisml_tenant (str | Unset):
        body (DataVolumePatchRequest):  Example: {'description': 'Shared raw datasets directory
            (expanded).', 'size': '4Ti'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DataVolume | Problem]
    """

    kwargs = _get_kwargs(
        name=name,
        body=body,
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
    body: DataVolumePatchRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> DataVolume | Problem | None:
    """Expand or relabel a data volume

    Args:
        name (str):
        x_axisml_tenant (str | Unset):
        body (DataVolumePatchRequest):  Example: {'description': 'Shared raw datasets directory
            (expanded).', 'size': '4Ti'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DataVolume | Problem
    """

    return sync_detailed(
        name=name,
        client=client,
        body=body,
        x_axisml_tenant=x_axisml_tenant,
    ).parsed


async def asyncio_detailed(
    name: str,
    *,
    client: AuthenticatedClient | Client,
    body: DataVolumePatchRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[DataVolume | Problem]:
    """Expand or relabel a data volume

    Args:
        name (str):
        x_axisml_tenant (str | Unset):
        body (DataVolumePatchRequest):  Example: {'description': 'Shared raw datasets directory
            (expanded).', 'size': '4Ti'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[DataVolume | Problem]
    """

    kwargs = _get_kwargs(
        name=name,
        body=body,
        x_axisml_tenant=x_axisml_tenant,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    name: str,
    *,
    client: AuthenticatedClient | Client,
    body: DataVolumePatchRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> DataVolume | Problem | None:
    """Expand or relabel a data volume

    Args:
        name (str):
        x_axisml_tenant (str | Unset):
        body (DataVolumePatchRequest):  Example: {'description': 'Shared raw datasets directory
            (expanded).', 'size': '4Ti'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        DataVolume | Problem
    """

    return (
        await asyncio_detailed(
            name=name,
            client=client,
            body=body,
            x_axisml_tenant=x_axisml_tenant,
        )
    ).parsed
