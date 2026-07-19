from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.compute_service_error import ComputeServiceError
from ...models.ml_run import MLRun
from ...models.ml_run_create_request import MLRunCreateRequest
from ...types import Response


def _get_kwargs(
    namespace: str,
    *,
    body: MLRunCreateRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/namespaces/{namespace}/mlruns".format(
            namespace=quote(str(namespace), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ComputeServiceError | MLRun:
    if response.status_code == 201:
        response_201 = MLRun.from_dict(response.json())

        return response_201

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
) -> Response[ComputeServiceError | MLRun]:
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
    body: MLRunCreateRequest,
) -> Response[ComputeServiceError | MLRun]:
    r"""Submit an MLRun

    Args:
        namespace (str):
        body (MLRunCreateRequest):  Example: {'backend': {'engine': 'pytorchjob', 'name':
            'kubeflow-trainer'}, 'configMaps': [{'data': {'trainer.yaml': 'epochs: 90\nbatchSize:
            256\n'}, 'name': 'trainer-config'}], 'description': 'Distributed ResNet-50 training on
            ImageNet.', 'displayName': 'ResNet-50 Training #7', 'labels': {'team': 'vision'}, 'name':
            'resnet-train-7', 'poolName': 'gpu-a100', 'roles': [{'name': 'worker', 'replicas': 4,
            'restartPolicy': 'OnFailure', 'template': {'args': ['--epochs', '90', '--batch-size',
            '256'], 'command': ['python', 'train.py'], 'env': [{'name': 'NCCL_DEBUG', 'value':
            'INFO'}], 'image': 'registry.axisml.io/training/resnet:1.4.0', 'resources': {'limits':
            {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}, 'requests': {'cpu': '8', 'memory':
            '64Gi', 'nvidia.com/gpu': '2'}}}}], 'runPolicy': {'activeDeadlineSeconds': 86400,
            'backoffLimit': 2, 'ttlSecondsAfterFinished': 3600}, 'unitName': 'a100-2x'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComputeServiceError | MLRun]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    namespace: str,
    *,
    client: AuthenticatedClient | Client,
    body: MLRunCreateRequest,
) -> ComputeServiceError | MLRun | None:
    r"""Submit an MLRun

    Args:
        namespace (str):
        body (MLRunCreateRequest):  Example: {'backend': {'engine': 'pytorchjob', 'name':
            'kubeflow-trainer'}, 'configMaps': [{'data': {'trainer.yaml': 'epochs: 90\nbatchSize:
            256\n'}, 'name': 'trainer-config'}], 'description': 'Distributed ResNet-50 training on
            ImageNet.', 'displayName': 'ResNet-50 Training #7', 'labels': {'team': 'vision'}, 'name':
            'resnet-train-7', 'poolName': 'gpu-a100', 'roles': [{'name': 'worker', 'replicas': 4,
            'restartPolicy': 'OnFailure', 'template': {'args': ['--epochs', '90', '--batch-size',
            '256'], 'command': ['python', 'train.py'], 'env': [{'name': 'NCCL_DEBUG', 'value':
            'INFO'}], 'image': 'registry.axisml.io/training/resnet:1.4.0', 'resources': {'limits':
            {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}, 'requests': {'cpu': '8', 'memory':
            '64Gi', 'nvidia.com/gpu': '2'}}}}], 'runPolicy': {'activeDeadlineSeconds': 86400,
            'backoffLimit': 2, 'ttlSecondsAfterFinished': 3600}, 'unitName': 'a100-2x'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComputeServiceError | MLRun
    """

    return sync_detailed(
        namespace=namespace,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    namespace: str,
    *,
    client: AuthenticatedClient | Client,
    body: MLRunCreateRequest,
) -> Response[ComputeServiceError | MLRun]:
    r"""Submit an MLRun

    Args:
        namespace (str):
        body (MLRunCreateRequest):  Example: {'backend': {'engine': 'pytorchjob', 'name':
            'kubeflow-trainer'}, 'configMaps': [{'data': {'trainer.yaml': 'epochs: 90\nbatchSize:
            256\n'}, 'name': 'trainer-config'}], 'description': 'Distributed ResNet-50 training on
            ImageNet.', 'displayName': 'ResNet-50 Training #7', 'labels': {'team': 'vision'}, 'name':
            'resnet-train-7', 'poolName': 'gpu-a100', 'roles': [{'name': 'worker', 'replicas': 4,
            'restartPolicy': 'OnFailure', 'template': {'args': ['--epochs', '90', '--batch-size',
            '256'], 'command': ['python', 'train.py'], 'env': [{'name': 'NCCL_DEBUG', 'value':
            'INFO'}], 'image': 'registry.axisml.io/training/resnet:1.4.0', 'resources': {'limits':
            {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}, 'requests': {'cpu': '8', 'memory':
            '64Gi', 'nvidia.com/gpu': '2'}}}}], 'runPolicy': {'activeDeadlineSeconds': 86400,
            'backoffLimit': 2, 'ttlSecondsAfterFinished': 3600}, 'unitName': 'a100-2x'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComputeServiceError | MLRun]
    """

    kwargs = _get_kwargs(
        namespace=namespace,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    namespace: str,
    *,
    client: AuthenticatedClient | Client,
    body: MLRunCreateRequest,
) -> ComputeServiceError | MLRun | None:
    r"""Submit an MLRun

    Args:
        namespace (str):
        body (MLRunCreateRequest):  Example: {'backend': {'engine': 'pytorchjob', 'name':
            'kubeflow-trainer'}, 'configMaps': [{'data': {'trainer.yaml': 'epochs: 90\nbatchSize:
            256\n'}, 'name': 'trainer-config'}], 'description': 'Distributed ResNet-50 training on
            ImageNet.', 'displayName': 'ResNet-50 Training #7', 'labels': {'team': 'vision'}, 'name':
            'resnet-train-7', 'poolName': 'gpu-a100', 'roles': [{'name': 'worker', 'replicas': 4,
            'restartPolicy': 'OnFailure', 'template': {'args': ['--epochs', '90', '--batch-size',
            '256'], 'command': ['python', 'train.py'], 'env': [{'name': 'NCCL_DEBUG', 'value':
            'INFO'}], 'image': 'registry.axisml.io/training/resnet:1.4.0', 'resources': {'limits':
            {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}, 'requests': {'cpu': '8', 'memory':
            '64Gi', 'nvidia.com/gpu': '2'}}}}], 'runPolicy': {'activeDeadlineSeconds': 86400,
            'backoffLimit': 2, 'ttlSecondsAfterFinished': 3600}, 'unitName': 'a100-2x'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComputeServiceError | MLRun
    """

    return (
        await asyncio_detailed(
            namespace=namespace,
            client=client,
            body=body,
        )
    ).parsed
