from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.job import Job
from ...models.job_create_request import JobCreateRequest
from ...models.problem import Problem
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    body: JobCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    if not isinstance(x_axisml_tenant, Unset):
        headers["X-Axisml-Tenant"] = x_axisml_tenant

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/jobs",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Job | Problem | None:
    if response.status_code == 201:
        response_201 = Job.from_dict(response.json())

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
) -> Response[Job | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: JobCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[Job | Problem]:
    """Create a Job (reusable template)

    Args:
        x_axisml_tenant (str | Unset):
        body (JobCreateRequest):  Example: {'description': 'Distributed ResNet-50 training job on
            ImageNet.', 'displayName': 'ResNet-50 Training', 'labels': {'team': 'vision'}, 'name':
            'resnet-train', 'spec': {'artifacts': [{'kind': 'model', 'name': 'resnet50', 'version':
            '1.4.0'}], 'backend': {'engine': 'pytorchjob', 'name': 'native'}, 'poolName': 'gpu-a100',
            'roles': [{'name': 'worker', 'replicas': 4, 'restartPolicy': 'OnFailure', 'template':
            {'args': ['--epochs', '90', '--batch-size', '256'], 'command': ['python', 'train.py'],
            'env': [{'name': 'NCCL_DEBUG', 'value': 'INFO'}], 'image':
            'registry.axisml.io/training/resnet:1.4.0', 'ports': [{'containerPort': 8080, 'name':
            'http', 'protocol': 'TCP'}], 'resources': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu':
            '2'}, 'volumeMounts': [{'mountPath': '/data', 'name': 'data'}], 'volumes': [{'name':
            'data', 'persistentVolumeClaim': {'claimName': 'resnet-imagenet'}}]}}], 'runPolicy':
            {'activeDeadlineSeconds': 86400, 'backoffLimit': 2, 'progressDeadlineSeconds': 600,
            'ttlSecondsAfterFinished': 3600}, 'unitName': 'a100-2x'}}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Job | Problem]
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
    body: JobCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> Job | Problem | None:
    """Create a Job (reusable template)

    Args:
        x_axisml_tenant (str | Unset):
        body (JobCreateRequest):  Example: {'description': 'Distributed ResNet-50 training job on
            ImageNet.', 'displayName': 'ResNet-50 Training', 'labels': {'team': 'vision'}, 'name':
            'resnet-train', 'spec': {'artifacts': [{'kind': 'model', 'name': 'resnet50', 'version':
            '1.4.0'}], 'backend': {'engine': 'pytorchjob', 'name': 'native'}, 'poolName': 'gpu-a100',
            'roles': [{'name': 'worker', 'replicas': 4, 'restartPolicy': 'OnFailure', 'template':
            {'args': ['--epochs', '90', '--batch-size', '256'], 'command': ['python', 'train.py'],
            'env': [{'name': 'NCCL_DEBUG', 'value': 'INFO'}], 'image':
            'registry.axisml.io/training/resnet:1.4.0', 'ports': [{'containerPort': 8080, 'name':
            'http', 'protocol': 'TCP'}], 'resources': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu':
            '2'}, 'volumeMounts': [{'mountPath': '/data', 'name': 'data'}], 'volumes': [{'name':
            'data', 'persistentVolumeClaim': {'claimName': 'resnet-imagenet'}}]}}], 'runPolicy':
            {'activeDeadlineSeconds': 86400, 'backoffLimit': 2, 'progressDeadlineSeconds': 600,
            'ttlSecondsAfterFinished': 3600}, 'unitName': 'a100-2x'}}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Job | Problem
    """

    return sync_detailed(
        client=client,
        body=body,
        x_axisml_tenant=x_axisml_tenant,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: JobCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[Job | Problem]:
    """Create a Job (reusable template)

    Args:
        x_axisml_tenant (str | Unset):
        body (JobCreateRequest):  Example: {'description': 'Distributed ResNet-50 training job on
            ImageNet.', 'displayName': 'ResNet-50 Training', 'labels': {'team': 'vision'}, 'name':
            'resnet-train', 'spec': {'artifacts': [{'kind': 'model', 'name': 'resnet50', 'version':
            '1.4.0'}], 'backend': {'engine': 'pytorchjob', 'name': 'native'}, 'poolName': 'gpu-a100',
            'roles': [{'name': 'worker', 'replicas': 4, 'restartPolicy': 'OnFailure', 'template':
            {'args': ['--epochs', '90', '--batch-size', '256'], 'command': ['python', 'train.py'],
            'env': [{'name': 'NCCL_DEBUG', 'value': 'INFO'}], 'image':
            'registry.axisml.io/training/resnet:1.4.0', 'ports': [{'containerPort': 8080, 'name':
            'http', 'protocol': 'TCP'}], 'resources': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu':
            '2'}, 'volumeMounts': [{'mountPath': '/data', 'name': 'data'}], 'volumes': [{'name':
            'data', 'persistentVolumeClaim': {'claimName': 'resnet-imagenet'}}]}}], 'runPolicy':
            {'activeDeadlineSeconds': 86400, 'backoffLimit': 2, 'progressDeadlineSeconds': 600,
            'ttlSecondsAfterFinished': 3600}, 'unitName': 'a100-2x'}}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Job | Problem]
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
    body: JobCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> Job | Problem | None:
    """Create a Job (reusable template)

    Args:
        x_axisml_tenant (str | Unset):
        body (JobCreateRequest):  Example: {'description': 'Distributed ResNet-50 training job on
            ImageNet.', 'displayName': 'ResNet-50 Training', 'labels': {'team': 'vision'}, 'name':
            'resnet-train', 'spec': {'artifacts': [{'kind': 'model', 'name': 'resnet50', 'version':
            '1.4.0'}], 'backend': {'engine': 'pytorchjob', 'name': 'native'}, 'poolName': 'gpu-a100',
            'roles': [{'name': 'worker', 'replicas': 4, 'restartPolicy': 'OnFailure', 'template':
            {'args': ['--epochs', '90', '--batch-size', '256'], 'command': ['python', 'train.py'],
            'env': [{'name': 'NCCL_DEBUG', 'value': 'INFO'}], 'image':
            'registry.axisml.io/training/resnet:1.4.0', 'ports': [{'containerPort': 8080, 'name':
            'http', 'protocol': 'TCP'}], 'resources': {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu':
            '2'}, 'volumeMounts': [{'mountPath': '/data', 'name': 'data'}], 'volumes': [{'name':
            'data', 'persistentVolumeClaim': {'claimName': 'resnet-imagenet'}}]}}], 'runPolicy':
            {'activeDeadlineSeconds': 86400, 'backoffLimit': 2, 'progressDeadlineSeconds': 600,
            'ttlSecondsAfterFinished': 3600}, 'unitName': 'a100-2x'}}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Job | Problem
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
            x_axisml_tenant=x_axisml_tenant,
        )
    ).parsed
