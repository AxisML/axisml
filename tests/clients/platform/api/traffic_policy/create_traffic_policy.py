from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.problem import Problem
from ...models.traffic_policy import TrafficPolicy
from ...models.traffic_policy_create_request import TrafficPolicyCreateRequest
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    body: TrafficPolicyCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    if not isinstance(x_axisml_tenant, Unset):
        headers["X-Axisml-Tenant"] = x_axisml_tenant

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/trafficpolicies",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Problem | TrafficPolicy | None:
    if response.status_code == 201:
        response_201 = TrafficPolicy.from_dict(response.json())

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
) -> Response[Problem | TrafficPolicy]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: TrafficPolicyCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[Problem | TrafficPolicy]:
    """Create a traffic policy

    Args:
        x_axisml_tenant (str | Unset):
        body (TrafficPolicyCreateRequest):  Example: {'backends': [{'role': 'stable',
            'serviceName': 'resnet-serving-v1', 'weight': 90}, {'role': 'canary', 'serviceName':
            'resnet-serving-v2', 'weight': 10}], 'canaryPercent': 10, 'description': 'Canary traffic
            split for the ResNet-50 online inference service.', 'displayName': 'ResNet inference
            traffic', 'endpoint': {'hostname': 'infer.axisml.io', 'path': '/services/team-
            vision/resnet-serving/'}, 'mode': 'canary', 'name': 'resnet-serving'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TrafficPolicy]
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
    body: TrafficPolicyCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> Problem | TrafficPolicy | None:
    """Create a traffic policy

    Args:
        x_axisml_tenant (str | Unset):
        body (TrafficPolicyCreateRequest):  Example: {'backends': [{'role': 'stable',
            'serviceName': 'resnet-serving-v1', 'weight': 90}, {'role': 'canary', 'serviceName':
            'resnet-serving-v2', 'weight': 10}], 'canaryPercent': 10, 'description': 'Canary traffic
            split for the ResNet-50 online inference service.', 'displayName': 'ResNet inference
            traffic', 'endpoint': {'hostname': 'infer.axisml.io', 'path': '/services/team-
            vision/resnet-serving/'}, 'mode': 'canary', 'name': 'resnet-serving'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TrafficPolicy
    """

    return sync_detailed(
        client=client,
        body=body,
        x_axisml_tenant=x_axisml_tenant,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: TrafficPolicyCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[Problem | TrafficPolicy]:
    """Create a traffic policy

    Args:
        x_axisml_tenant (str | Unset):
        body (TrafficPolicyCreateRequest):  Example: {'backends': [{'role': 'stable',
            'serviceName': 'resnet-serving-v1', 'weight': 90}, {'role': 'canary', 'serviceName':
            'resnet-serving-v2', 'weight': 10}], 'canaryPercent': 10, 'description': 'Canary traffic
            split for the ResNet-50 online inference service.', 'displayName': 'ResNet inference
            traffic', 'endpoint': {'hostname': 'infer.axisml.io', 'path': '/services/team-
            vision/resnet-serving/'}, 'mode': 'canary', 'name': 'resnet-serving'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Problem | TrafficPolicy]
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
    body: TrafficPolicyCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> Problem | TrafficPolicy | None:
    """Create a traffic policy

    Args:
        x_axisml_tenant (str | Unset):
        body (TrafficPolicyCreateRequest):  Example: {'backends': [{'role': 'stable',
            'serviceName': 'resnet-serving-v1', 'weight': 90}, {'role': 'canary', 'serviceName':
            'resnet-serving-v2', 'weight': 10}], 'canaryPercent': 10, 'description': 'Canary traffic
            split for the ResNet-50 online inference service.', 'displayName': 'ResNet inference
            traffic', 'endpoint': {'hostname': 'infer.axisml.io', 'path': '/services/team-
            vision/resnet-serving/'}, 'mode': 'canary', 'name': 'resnet-serving'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Problem | TrafficPolicy
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
            x_axisml_tenant=x_axisml_tenant,
        )
    ).parsed
