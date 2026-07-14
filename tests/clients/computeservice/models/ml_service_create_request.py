from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.ml_service_backend import MLServiceBackend
    from ..models.ml_service_create_request_annotations import (
        MLServiceCreateRequestAnnotations,
    )
    from ..models.ml_service_create_request_labels import MLServiceCreateRequestLabels
    from ..models.ml_service_role_spec import MLServiceRoleSpec
    from ..models.ml_service_route import MLServiceRoute
    from ..models.ml_service_run_policy import MLServiceRunPolicy


T = TypeVar("T", bound="MLServiceCreateRequest")


@_attrs_define
class MLServiceCreateRequest:
    """
    Example:
        {'backend': {'engine': 'llminference', 'name': 'kserve'}, 'description': 'Llama-3 8B online inference on the
            vLLM backend.', 'displayName': 'Llama-3 8B inference service', 'kind': 'service', 'labels': {'team': 'vision'},
            'name': 'llama3-8b', 'poolName': 'gpu-a100', 'roles': [{'name': 'predictor', 'replicas': 2, 'template': {'args':
            ['--model', 'meta-llama/Llama-3-8b', '--max-model-len', '8192'], 'image':
            'registry.axisml.io/serving/vllm:0.6.2', 'ports': [{'containerPort': 8080, 'name': 'http', 'protocol': 'TCP'}],
            'resources': {'limits': {'cpu': '8', 'memory': '48Gi', 'nvidia.com/gpu': '1'}, 'requests': {'cpu': '8',
            'memory': '48Gi', 'nvidia.com/gpu': '1'}}}}], 'route': {'auth': {'jwt': {'issuer': 'https://auth.axisml.io',
            'jwksUri': 'https://auth.axisml.io/.well-known/jwks.json'}, 'type': 'jwt'}, 'enabled': True, 'hostname':
            'llama3-8b.team-vision.axisml.io', 'path': '/v1', 'portName': 'http', 'targetRole': 'predictor'}, 'runPolicy':
            {'progressDeadlineSeconds': 600}, 'unitName': 'a100-2x'}

    Attributes:
        name (str): MLService name, unique within the namespace.
        pool_name (str): Resource pool name resolved against the ResourcePool CRD via the Informer cache.
        roles (list[MLServiceRoleSpec]): Service topology roles (at least one).
        unit_name (str): Resource unit (shape) name within the selected pool.
        annotations (MLServiceCreateRequestAnnotations | Unset): User-defined annotations stored on the row and stamped
            onto the CR.
        backend (MLServiceBackend | None | Unset): Compute backend/engine that serves the workload; defaults to (native,
            deployment) when omitted.
        description (str | Unset): Free-text service description.
        display_name (str | Unset): Human-readable service label.
        kind (str | Unset): Service kind (service, workspace, tensorboard); immutable after create, defaults to service.
        labels (MLServiceCreateRequestLabels | Unset): User-defined labels stored on the row and stamped onto the CR.
        priority_class (str | Unset): Optional Kubernetes PriorityClass name for the service's pods.
        route (MLServiceRoute | None | Unset): Optional external entrypoint (HTTPRoute plus auth/rate-limit policies).
        run_policy (MLServiceRunPolicy | None | Unset): Service-level lifecycle controls (progress deadline).
    """

    name: str
    pool_name: str
    roles: list[MLServiceRoleSpec]
    unit_name: str
    annotations: MLServiceCreateRequestAnnotations | Unset = UNSET
    backend: MLServiceBackend | None | Unset = UNSET
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    kind: str | Unset = UNSET
    labels: MLServiceCreateRequestLabels | Unset = UNSET
    priority_class: str | Unset = UNSET
    route: MLServiceRoute | None | Unset = UNSET
    run_policy: MLServiceRunPolicy | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.ml_service_backend import MLServiceBackend
        from ..models.ml_service_route import MLServiceRoute
        from ..models.ml_service_run_policy import MLServiceRunPolicy

        name = self.name

        pool_name = self.pool_name

        roles = []
        for roles_item_data in self.roles:
            roles_item = roles_item_data.to_dict()
            roles.append(roles_item)

        unit_name = self.unit_name

        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        backend: dict[str, Any] | None | Unset
        if isinstance(self.backend, Unset):
            backend = UNSET
        elif isinstance(self.backend, MLServiceBackend):
            backend = self.backend.to_dict()
        else:
            backend = self.backend

        description = self.description

        display_name = self.display_name

        kind = self.kind

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        priority_class = self.priority_class

        route: dict[str, Any] | None | Unset
        if isinstance(self.route, Unset):
            route = UNSET
        elif isinstance(self.route, MLServiceRoute):
            route = self.route.to_dict()
        else:
            route = self.route

        run_policy: dict[str, Any] | None | Unset
        if isinstance(self.run_policy, Unset):
            run_policy = UNSET
        elif isinstance(self.run_policy, MLServiceRunPolicy):
            run_policy = self.run_policy.to_dict()
        else:
            run_policy = self.run_policy

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
                "poolName": pool_name,
                "roles": roles,
                "unitName": unit_name,
            }
        )
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if backend is not UNSET:
            field_dict["backend"] = backend
        if description is not UNSET:
            field_dict["description"] = description
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if kind is not UNSET:
            field_dict["kind"] = kind
        if labels is not UNSET:
            field_dict["labels"] = labels
        if priority_class is not UNSET:
            field_dict["priorityClass"] = priority_class
        if route is not UNSET:
            field_dict["route"] = route
        if run_policy is not UNSET:
            field_dict["runPolicy"] = run_policy

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.ml_service_backend import MLServiceBackend
        from ..models.ml_service_create_request_annotations import (
            MLServiceCreateRequestAnnotations,
        )
        from ..models.ml_service_create_request_labels import (
            MLServiceCreateRequestLabels,
        )
        from ..models.ml_service_role_spec import MLServiceRoleSpec
        from ..models.ml_service_route import MLServiceRoute
        from ..models.ml_service_run_policy import MLServiceRunPolicy

        d = dict(src_dict)
        name = d.pop("name")

        pool_name = d.pop("poolName")

        roles = []
        _roles = d.pop("roles")
        for roles_item_data in _roles:
            roles_item = MLServiceRoleSpec.from_dict(roles_item_data)

            roles.append(roles_item)

        unit_name = d.pop("unitName")

        _annotations = d.pop("annotations", UNSET)
        annotations: MLServiceCreateRequestAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = MLServiceCreateRequestAnnotations.from_dict(_annotations)

        def _parse_backend(data: object) -> MLServiceBackend | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                backend_type_1 = MLServiceBackend.from_dict(data)

                return backend_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(MLServiceBackend | None | Unset, data)

        backend = _parse_backend(d.pop("backend", UNSET))

        description = d.pop("description", UNSET)

        display_name = d.pop("displayName", UNSET)

        kind = d.pop("kind", UNSET)

        _labels = d.pop("labels", UNSET)
        labels: MLServiceCreateRequestLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = MLServiceCreateRequestLabels.from_dict(_labels)

        priority_class = d.pop("priorityClass", UNSET)

        def _parse_route(data: object) -> MLServiceRoute | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                route_type_1 = MLServiceRoute.from_dict(data)

                return route_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(MLServiceRoute | None | Unset, data)

        route = _parse_route(d.pop("route", UNSET))

        def _parse_run_policy(data: object) -> MLServiceRunPolicy | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                run_policy_type_1 = MLServiceRunPolicy.from_dict(data)

                return run_policy_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(MLServiceRunPolicy | None | Unset, data)

        run_policy = _parse_run_policy(d.pop("runPolicy", UNSET))

        ml_service_create_request = cls(
            name=name,
            pool_name=pool_name,
            roles=roles,
            unit_name=unit_name,
            annotations=annotations,
            backend=backend,
            description=description,
            display_name=display_name,
            kind=kind,
            labels=labels,
            priority_class=priority_class,
            route=route,
            run_policy=run_policy,
        )

        ml_service_create_request.additional_properties = d
        return ml_service_create_request

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
