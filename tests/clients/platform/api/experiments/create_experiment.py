from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.experiment import Experiment
from ...models.experiment_create_request import ExperimentCreateRequest
from ...models.problem import Problem
from ...types import UNSET, Response, Unset


def _get_kwargs(
    *,
    body: ExperimentCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    if not isinstance(x_axisml_tenant, Unset):
        headers["X-Axisml-Tenant"] = x_axisml_tenant

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/experiments",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Experiment | Problem | None:
    if response.status_code == 201:
        response_201 = Experiment.from_dict(response.json())

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
) -> Response[Experiment | Problem]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: ExperimentCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[Experiment | Problem]:
    """Create an experiment (training template)

    Args:
        x_axisml_tenant (str | Unset):
        body (ExperimentCreateRequest):  Example: {'description': 'Training experiment fine-tuning
            BERT on a Chinese corpus.', 'displayName': 'BERT fine-tuning experiment', 'labels':
            {'team': 'nlp'}, 'name': 'bert-finetune', 'spec': {'artifacts': [{'kind': 'model', 'name':
            'bert-base', 'version': '2.1.0'}], 'backend': {'engine': 'pytorchjob', 'name': 'kubeflow-
            trainer'}, 'poolName': 'gpu-a100', 'roles': [{'name': 'worker', 'replicas': 2,
            'restartPolicy': 'OnFailure', 'template': {'args': ['--lr', '0.001', '--epochs', '10', '--
            batch-size', '64'], 'command': ['python', 'train.py'], 'env': [{'name': 'WANDB_MODE',
            'value': 'offline'}], 'image': 'registry.axisml.io/training/bert:2.1.0', 'resources':
            {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}}}], 'runPolicy':
            {'activeDeadlineSeconds': 86400, 'backoffLimit': 2, 'ttlSecondsAfterFinished': 3600},
            'unitName': 'a100-2x'}}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Experiment | Problem]
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
    body: ExperimentCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> Experiment | Problem | None:
    """Create an experiment (training template)

    Args:
        x_axisml_tenant (str | Unset):
        body (ExperimentCreateRequest):  Example: {'description': 'Training experiment fine-tuning
            BERT on a Chinese corpus.', 'displayName': 'BERT fine-tuning experiment', 'labels':
            {'team': 'nlp'}, 'name': 'bert-finetune', 'spec': {'artifacts': [{'kind': 'model', 'name':
            'bert-base', 'version': '2.1.0'}], 'backend': {'engine': 'pytorchjob', 'name': 'kubeflow-
            trainer'}, 'poolName': 'gpu-a100', 'roles': [{'name': 'worker', 'replicas': 2,
            'restartPolicy': 'OnFailure', 'template': {'args': ['--lr', '0.001', '--epochs', '10', '--
            batch-size', '64'], 'command': ['python', 'train.py'], 'env': [{'name': 'WANDB_MODE',
            'value': 'offline'}], 'image': 'registry.axisml.io/training/bert:2.1.0', 'resources':
            {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}}}], 'runPolicy':
            {'activeDeadlineSeconds': 86400, 'backoffLimit': 2, 'ttlSecondsAfterFinished': 3600},
            'unitName': 'a100-2x'}}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Experiment | Problem
    """

    return sync_detailed(
        client=client,
        body=body,
        x_axisml_tenant=x_axisml_tenant,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: ExperimentCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> Response[Experiment | Problem]:
    """Create an experiment (training template)

    Args:
        x_axisml_tenant (str | Unset):
        body (ExperimentCreateRequest):  Example: {'description': 'Training experiment fine-tuning
            BERT on a Chinese corpus.', 'displayName': 'BERT fine-tuning experiment', 'labels':
            {'team': 'nlp'}, 'name': 'bert-finetune', 'spec': {'artifacts': [{'kind': 'model', 'name':
            'bert-base', 'version': '2.1.0'}], 'backend': {'engine': 'pytorchjob', 'name': 'kubeflow-
            trainer'}, 'poolName': 'gpu-a100', 'roles': [{'name': 'worker', 'replicas': 2,
            'restartPolicy': 'OnFailure', 'template': {'args': ['--lr', '0.001', '--epochs', '10', '--
            batch-size', '64'], 'command': ['python', 'train.py'], 'env': [{'name': 'WANDB_MODE',
            'value': 'offline'}], 'image': 'registry.axisml.io/training/bert:2.1.0', 'resources':
            {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}}}], 'runPolicy':
            {'activeDeadlineSeconds': 86400, 'backoffLimit': 2, 'ttlSecondsAfterFinished': 3600},
            'unitName': 'a100-2x'}}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Experiment | Problem]
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
    body: ExperimentCreateRequest,
    x_axisml_tenant: str | Unset = UNSET,
) -> Experiment | Problem | None:
    """Create an experiment (training template)

    Args:
        x_axisml_tenant (str | Unset):
        body (ExperimentCreateRequest):  Example: {'description': 'Training experiment fine-tuning
            BERT on a Chinese corpus.', 'displayName': 'BERT fine-tuning experiment', 'labels':
            {'team': 'nlp'}, 'name': 'bert-finetune', 'spec': {'artifacts': [{'kind': 'model', 'name':
            'bert-base', 'version': '2.1.0'}], 'backend': {'engine': 'pytorchjob', 'name': 'kubeflow-
            trainer'}, 'poolName': 'gpu-a100', 'roles': [{'name': 'worker', 'replicas': 2,
            'restartPolicy': 'OnFailure', 'template': {'args': ['--lr', '0.001', '--epochs', '10', '--
            batch-size', '64'], 'command': ['python', 'train.py'], 'env': [{'name': 'WANDB_MODE',
            'value': 'offline'}], 'image': 'registry.axisml.io/training/bert:2.1.0', 'resources':
            {'cpu': '8', 'memory': '64Gi', 'nvidia.com/gpu': '2'}}}], 'runPolicy':
            {'activeDeadlineSeconds': 86400, 'backoffLimit': 2, 'ttlSecondsAfterFinished': 3600},
            'unitName': 'a100-2x'}}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Experiment | Problem
    """

    return (
        await asyncio_detailed(
            client=client,
            body=body,
            x_axisml_tenant=x_axisml_tenant,
        )
    ).parsed
