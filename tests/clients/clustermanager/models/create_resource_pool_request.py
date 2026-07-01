from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_toleration import Corev1Toleration
    from ..models.create_resource_pool_request_annotations import (
        CreateResourcePoolRequestAnnotations,
    )
    from ..models.create_resource_pool_request_labels import (
        CreateResourcePoolRequestLabels,
    )
    from ..models.create_resource_pool_request_node_selector import (
        CreateResourcePoolRequestNodeSelector,
    )
    from ..models.server_create_resource_unit_request import (
        ServerCreateResourceUnitRequest,
    )


T = TypeVar("T", bound="CreateResourcePoolRequest")


@_attrs_define
class CreateResourcePoolRequest:
    """
    Example:
        {'description': 'A100 GPU resource pool.', 'labels': {'tier': 'gpu'}, 'name': 'gpu-a100', 'nodeSelector':
            {'axisml.io/gpu': 'a100'}, 'tolerations': [{'effect': 'NoSchedule', 'key': 'nvidia.com/gpu', 'operator':
            'Exists'}], 'units': [{'description': '2× A100 GPU compute unit.', 'limits': {'cpu': '16', 'memory': '128Gi',
            'nvidia.com/gpu': '2'}, 'name': 'a100-2x', 'nodeSelector': {'axisml.io/gpu': 'a100'}, 'requests': {'cpu': '16',
            'memory': '128Gi', 'nvidia.com/gpu': '2'}}]}

    Attributes:
        annotations (CreateResourcePoolRequestAnnotations | Unset): User-defined annotations to set on the pool.
        description (str | Unset): Human-readable description of the pool.
        labels (CreateResourcePoolRequestLabels | Unset): User-defined labels to set on the pool.
        name (str | Unset): Pool name to create; must be unique and DNS-1123 compliant.
        node_selector (CreateResourcePoolRequestNodeSelector | Unset): Node labels that workloads scheduled into this
            pool must match.
        tolerations (list[Corev1Toleration] | Unset): Tolerations applied to workloads scheduled into this pool.
        units (list[ServerCreateResourceUnitRequest] | Unset): Initial set of resource units to create inline with the
            pool.
    """

    annotations: CreateResourcePoolRequestAnnotations | Unset = UNSET
    description: str | Unset = UNSET
    labels: CreateResourcePoolRequestLabels | Unset = UNSET
    name: str | Unset = UNSET
    node_selector: CreateResourcePoolRequestNodeSelector | Unset = UNSET
    tolerations: list[Corev1Toleration] | Unset = UNSET
    units: list[ServerCreateResourceUnitRequest] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        description = self.description

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        name = self.name

        node_selector: dict[str, Any] | Unset = UNSET
        if not isinstance(self.node_selector, Unset):
            node_selector = self.node_selector.to_dict()

        tolerations: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.tolerations, Unset):
            tolerations = []
            for tolerations_item_data in self.tolerations:
                tolerations_item = tolerations_item_data.to_dict()
                tolerations.append(tolerations_item)

        units: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.units, Unset):
            units = []
            for units_item_data in self.units:
                units_item = units_item_data.to_dict()
                units.append(units_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if description is not UNSET:
            field_dict["description"] = description
        if labels is not UNSET:
            field_dict["labels"] = labels
        if name is not UNSET:
            field_dict["name"] = name
        if node_selector is not UNSET:
            field_dict["nodeSelector"] = node_selector
        if tolerations is not UNSET:
            field_dict["tolerations"] = tolerations
        if units is not UNSET:
            field_dict["units"] = units

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_toleration import Corev1Toleration
        from ..models.create_resource_pool_request_annotations import (
            CreateResourcePoolRequestAnnotations,
        )
        from ..models.create_resource_pool_request_labels import (
            CreateResourcePoolRequestLabels,
        )
        from ..models.create_resource_pool_request_node_selector import (
            CreateResourcePoolRequestNodeSelector,
        )
        from ..models.server_create_resource_unit_request import (
            ServerCreateResourceUnitRequest,
        )

        d = dict(src_dict)
        _annotations = d.pop("annotations", UNSET)
        annotations: CreateResourcePoolRequestAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = CreateResourcePoolRequestAnnotations.from_dict(_annotations)

        description = d.pop("description", UNSET)

        _labels = d.pop("labels", UNSET)
        labels: CreateResourcePoolRequestLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = CreateResourcePoolRequestLabels.from_dict(_labels)

        name = d.pop("name", UNSET)

        _node_selector = d.pop("nodeSelector", UNSET)
        node_selector: CreateResourcePoolRequestNodeSelector | Unset
        if isinstance(_node_selector, Unset):
            node_selector = UNSET
        else:
            node_selector = CreateResourcePoolRequestNodeSelector.from_dict(
                _node_selector
            )

        _tolerations = d.pop("tolerations", UNSET)
        tolerations: list[Corev1Toleration] | Unset = UNSET
        if _tolerations is not UNSET:
            tolerations = []
            for tolerations_item_data in _tolerations:
                tolerations_item = Corev1Toleration.from_dict(tolerations_item_data)

                tolerations.append(tolerations_item)

        _units = d.pop("units", UNSET)
        units: list[ServerCreateResourceUnitRequest] | Unset = UNSET
        if _units is not UNSET:
            units = []
            for units_item_data in _units:
                units_item = ServerCreateResourceUnitRequest.from_dict(units_item_data)

                units.append(units_item)

        create_resource_pool_request = cls(
            annotations=annotations,
            description=description,
            labels=labels,
            name=name,
            node_selector=node_selector,
            tolerations=tolerations,
            units=units,
        )

        create_resource_pool_request.additional_properties = d
        return create_resource_pool_request

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
