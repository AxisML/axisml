from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.compute_service_error import ComputeServiceError
from ...models.ml_service_list import MLServiceList
from ...types import UNSET, Response, Unset


def _get_kwargs(
    namespace: str,
    *,
    kind: str | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    label_selector: str | Unset = UNSET,
) -> dict[str, Any]:
    params: dict[str, Any] = {}

    params["kind"] = kind

    params["limit"] = limit

    params["continue"] = continue_

    params["labelSelector"] = label_selector

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/v1/namespaces/{namespace}/mlservices".format(
            namespace=quote(str(namespace), safe=""),
        ),
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ComputeServiceError | MLServiceList:
    if response.status_code == 200:
        response_200 = MLServiceList.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = ComputeServiceError.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = ComputeServiceError.from_dict(response.json())

        return response_401

    if response.status_code == 403:
        response_403 = ComputeServiceError.from_dict(response.json())

        return response_403

    if response.status_code == 404:
        response_404 = ComputeServiceError.from_dict(response.json())

        return response_404

    if response.status_code == 409:
        response_409 = ComputeServiceError.from_dict(response.json())

        return response_409

    if response.status_code == 412:
        response_412 = ComputeServiceError.from_dict(response.json())

        return response_412

    if response.status_code == 422:
        response_422 = ComputeServiceError.from_dict(response.json())

        return response_422

    if response.status_code == 503:
        response_503 = ComputeServiceError.from_dict(response.json())

        return response_503

    response_default = ComputeServiceError.from_dict(response.json())

    return response_default


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[ComputeServiceError | MLServiceList]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    namespace: str,
    *,
    client: AuthenticatedClient | Client,
    kind: str | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    label_selector: str | Unset = UNSET,
) -> Response[ComputeServiceError | MLServiceList]:
    """List MLServices in a namespace

    Args:
        namespace (str):
        kind (str | Unset):
        limit (int | Unset):
        continue_ (str | Unset):
        label_selector (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComputeServiceError | MLServiceList]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        kind=kind,
        limit=limit,
        continue_=continue_,
        label_selector=label_selector,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    namespace: str,
    *,
    client: AuthenticatedClient | Client,
    kind: str | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    label_selector: str | Unset = UNSET,
) -> ComputeServiceError | MLServiceList | None:
    """List MLServices in a namespace

    Args:
        namespace (str):
        kind (str | Unset):
        limit (int | Unset):
        continue_ (str | Unset):
        label_selector (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComputeServiceError | MLServiceList
    """

    return sync_detailed(
        namespace=namespace,
        client=client,
        kind=kind,
        limit=limit,
        continue_=continue_,
        label_selector=label_selector,
    ).parsed


async def asyncio_detailed(
    namespace: str,
    *,
    client: AuthenticatedClient | Client,
    kind: str | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    label_selector: str | Unset = UNSET,
) -> Response[ComputeServiceError | MLServiceList]:
    """List MLServices in a namespace

    Args:
        namespace (str):
        kind (str | Unset):
        limit (int | Unset):
        continue_ (str | Unset):
        label_selector (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComputeServiceError | MLServiceList]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        kind=kind,
        limit=limit,
        continue_=continue_,
        label_selector=label_selector,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    namespace: str,
    *,
    client: AuthenticatedClient | Client,
    kind: str | Unset = UNSET,
    limit: int | Unset = UNSET,
    continue_: str | Unset = UNSET,
    label_selector: str | Unset = UNSET,
) -> ComputeServiceError | MLServiceList | None:
    """List MLServices in a namespace

    Args:
        namespace (str):
        kind (str | Unset):
        limit (int | Unset):
        continue_ (str | Unset):
        label_selector (str | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComputeServiceError | MLServiceList
    """

    return (
        await asyncio_detailed(
            namespace=namespace,
            client=client,
            kind=kind,
            limit=limit,
            continue_=continue_,
            label_selector=label_selector,
        )
    ).parsed
