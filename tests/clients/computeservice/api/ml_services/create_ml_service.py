from http import HTTPStatus
from typing import Any
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...models.compute_service_error import ComputeServiceError
from ...models.ml_service import MLService
from ...models.ml_service_create_request import MLServiceCreateRequest
from ...types import Response


def _get_kwargs(
    namespace: str,
    *,
    body: MLServiceCreateRequest,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/v1/namespaces/{namespace}/mlservices".format(
            namespace=quote(str(namespace), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> ComputeServiceError | MLService:
    if response.status_code == 201:
        response_201 = MLService.from_dict(response.json())

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
) -> Response[ComputeServiceError | MLService]:
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
    body: MLServiceCreateRequest,
) -> Response[ComputeServiceError | MLService]:
    r"""Submit an MLService

    Args:
        namespace (str):
        body (MLServiceCreateRequest):  Example: {'backend': {'engine': 'llminference', 'name':
            'kserve'}, 'configMaps': [{'data': {'config.yaml': 'model: llama3-8b\n'}, 'name':
            'serving-config'}], 'description': 'Llama-3 8B online inference on the vLLM backend.',
            'displayName': 'Llama-3 8B inference service', 'kind': 'service', 'labels': {'team':
            'vision'}, 'name': 'llama3-8b', 'poolName': 'gpu-a100', 'roles': [{'name': 'predictor',
            'replicas': 2, 'template': {'args': ['--model', 'meta-llama/Llama-3-8b', '--max-model-
            len', '8192'], 'image': 'registry.axisml.io/serving/vllm:0.6.2', 'ports':
            [{'containerPort': 8080, 'name': 'http', 'protocol': 'TCP'}], 'resources': {'limits':
            {'cpu': '8', 'memory': '48Gi', 'nvidia.com/gpu': '1'}, 'requests': {'cpu': '8', 'memory':
            '48Gi', 'nvidia.com/gpu': '1'}}}}], 'route': {'auth': {'jwt': {'issuer':
            'https://auth.axisml.io', 'jwksUri': 'https://auth.axisml.io/.well-known/jwks.json'},
            'type': 'jwt'}, 'enabled': True, 'hostname': 'llama3-8b.team-vision.axisml.io', 'path':
            '/v1', 'portName': 'http', 'targetRole': 'predictor'}, 'runPolicy':
            {'progressDeadlineSeconds': 600}, 'unitName': 'a100-2x'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComputeServiceError | MLService]
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
    body: MLServiceCreateRequest,
) -> ComputeServiceError | MLService | None:
    r"""Submit an MLService

    Args:
        namespace (str):
        body (MLServiceCreateRequest):  Example: {'backend': {'engine': 'llminference', 'name':
            'kserve'}, 'configMaps': [{'data': {'config.yaml': 'model: llama3-8b\n'}, 'name':
            'serving-config'}], 'description': 'Llama-3 8B online inference on the vLLM backend.',
            'displayName': 'Llama-3 8B inference service', 'kind': 'service', 'labels': {'team':
            'vision'}, 'name': 'llama3-8b', 'poolName': 'gpu-a100', 'roles': [{'name': 'predictor',
            'replicas': 2, 'template': {'args': ['--model', 'meta-llama/Llama-3-8b', '--max-model-
            len', '8192'], 'image': 'registry.axisml.io/serving/vllm:0.6.2', 'ports':
            [{'containerPort': 8080, 'name': 'http', 'protocol': 'TCP'}], 'resources': {'limits':
            {'cpu': '8', 'memory': '48Gi', 'nvidia.com/gpu': '1'}, 'requests': {'cpu': '8', 'memory':
            '48Gi', 'nvidia.com/gpu': '1'}}}}], 'route': {'auth': {'jwt': {'issuer':
            'https://auth.axisml.io', 'jwksUri': 'https://auth.axisml.io/.well-known/jwks.json'},
            'type': 'jwt'}, 'enabled': True, 'hostname': 'llama3-8b.team-vision.axisml.io', 'path':
            '/v1', 'portName': 'http', 'targetRole': 'predictor'}, 'runPolicy':
            {'progressDeadlineSeconds': 600}, 'unitName': 'a100-2x'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComputeServiceError | MLService
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
    body: MLServiceCreateRequest,
) -> Response[ComputeServiceError | MLService]:
    r"""Submit an MLService

    Args:
        namespace (str):
        body (MLServiceCreateRequest):  Example: {'backend': {'engine': 'llminference', 'name':
            'kserve'}, 'configMaps': [{'data': {'config.yaml': 'model: llama3-8b\n'}, 'name':
            'serving-config'}], 'description': 'Llama-3 8B online inference on the vLLM backend.',
            'displayName': 'Llama-3 8B inference service', 'kind': 'service', 'labels': {'team':
            'vision'}, 'name': 'llama3-8b', 'poolName': 'gpu-a100', 'roles': [{'name': 'predictor',
            'replicas': 2, 'template': {'args': ['--model', 'meta-llama/Llama-3-8b', '--max-model-
            len', '8192'], 'image': 'registry.axisml.io/serving/vllm:0.6.2', 'ports':
            [{'containerPort': 8080, 'name': 'http', 'protocol': 'TCP'}], 'resources': {'limits':
            {'cpu': '8', 'memory': '48Gi', 'nvidia.com/gpu': '1'}, 'requests': {'cpu': '8', 'memory':
            '48Gi', 'nvidia.com/gpu': '1'}}}}], 'route': {'auth': {'jwt': {'issuer':
            'https://auth.axisml.io', 'jwksUri': 'https://auth.axisml.io/.well-known/jwks.json'},
            'type': 'jwt'}, 'enabled': True, 'hostname': 'llama3-8b.team-vision.axisml.io', 'path':
            '/v1', 'portName': 'http', 'targetRole': 'predictor'}, 'runPolicy':
            {'progressDeadlineSeconds': 600}, 'unitName': 'a100-2x'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[ComputeServiceError | MLService]
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
    body: MLServiceCreateRequest,
) -> ComputeServiceError | MLService | None:
    r"""Submit an MLService

    Args:
        namespace (str):
        body (MLServiceCreateRequest):  Example: {'backend': {'engine': 'llminference', 'name':
            'kserve'}, 'configMaps': [{'data': {'config.yaml': 'model: llama3-8b\n'}, 'name':
            'serving-config'}], 'description': 'Llama-3 8B online inference on the vLLM backend.',
            'displayName': 'Llama-3 8B inference service', 'kind': 'service', 'labels': {'team':
            'vision'}, 'name': 'llama3-8b', 'poolName': 'gpu-a100', 'roles': [{'name': 'predictor',
            'replicas': 2, 'template': {'args': ['--model', 'meta-llama/Llama-3-8b', '--max-model-
            len', '8192'], 'image': 'registry.axisml.io/serving/vllm:0.6.2', 'ports':
            [{'containerPort': 8080, 'name': 'http', 'protocol': 'TCP'}], 'resources': {'limits':
            {'cpu': '8', 'memory': '48Gi', 'nvidia.com/gpu': '1'}, 'requests': {'cpu': '8', 'memory':
            '48Gi', 'nvidia.com/gpu': '1'}}}}], 'route': {'auth': {'jwt': {'issuer':
            'https://auth.axisml.io', 'jwksUri': 'https://auth.axisml.io/.well-known/jwks.json'},
            'type': 'jwt'}, 'enabled': True, 'hostname': 'llama3-8b.team-vision.axisml.io', 'path':
            '/v1', 'portName': 'http', 'targetRole': 'predictor'}, 'runPolicy':
            {'progressDeadlineSeconds': 600}, 'unitName': 'a100-2x'}.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        ComputeServiceError | MLService
    """

    return (
        await asyncio_detailed(
            namespace=namespace,
            client=client,
            body=body,
        )
    ).parsed
