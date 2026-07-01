from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_toleration import Corev1Toleration
    from ..models.patch_resource_pool_request_annotations import (
        PatchResourcePoolRequestAnnotations,
    )
    from ..models.patch_resource_pool_request_labels import (
        PatchResourcePoolRequestLabels,
    )
    from ..models.patch_resource_pool_request_node_selector import (
        PatchResourcePoolRequestNodeSelector,
    )


T = TypeVar("T", bound="PatchResourcePoolRequest")


@_attrs_define
class PatchResourcePoolRequest:
    """
    Example:
        {'description': 'A100 GPU resource pool (updated).', 'labels': {'region': 'cn-east', 'tier': 'gpu'}}

    Attributes:
        annotations (PatchResourcePoolRequestAnnotations | Unset): Replacement annotations for the pool.
        description (None | str | Unset): New description; omit to leave unchanged, empty string to clear.
        labels (PatchResourcePoolRequestLabels | Unset): Replacement labels for the pool.
        node_selector (PatchResourcePoolRequestNodeSelector | Unset): Replacement node selector for the pool.
        tolerations (list[Corev1Toleration] | Unset): Replacement tolerations for the pool.
    """

    annotations: PatchResourcePoolRequestAnnotations | Unset = UNSET
    description: None | str | Unset = UNSET
    labels: PatchResourcePoolRequestLabels | Unset = UNSET
    node_selector: PatchResourcePoolRequestNodeSelector | Unset = UNSET
    tolerations: list[Corev1Toleration] | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        description: None | str | Unset
        if isinstance(self.description, Unset):
            description = UNSET
        else:
            description = self.description

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        node_selector: dict[str, Any] | Unset = UNSET
        if not isinstance(self.node_selector, Unset):
            node_selector = self.node_selector.to_dict()

        tolerations: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.tolerations, Unset):
            tolerations = []
            for tolerations_item_data in self.tolerations:
                tolerations_item = tolerations_item_data.to_dict()
                tolerations.append(tolerations_item)

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if description is not UNSET:
            field_dict["description"] = description
        if labels is not UNSET:
            field_dict["labels"] = labels
        if node_selector is not UNSET:
            field_dict["nodeSelector"] = node_selector
        if tolerations is not UNSET:
            field_dict["tolerations"] = tolerations

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_toleration import Corev1Toleration
        from ..models.patch_resource_pool_request_annotations import (
            PatchResourcePoolRequestAnnotations,
        )
        from ..models.patch_resource_pool_request_labels import (
            PatchResourcePoolRequestLabels,
        )
        from ..models.patch_resource_pool_request_node_selector import (
            PatchResourcePoolRequestNodeSelector,
        )

        d = dict(src_dict)
        _annotations = d.pop("annotations", UNSET)
        annotations: PatchResourcePoolRequestAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = PatchResourcePoolRequestAnnotations.from_dict(_annotations)

        def _parse_description(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        description = _parse_description(d.pop("description", UNSET))

        _labels = d.pop("labels", UNSET)
        labels: PatchResourcePoolRequestLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = PatchResourcePoolRequestLabels.from_dict(_labels)

        _node_selector = d.pop("nodeSelector", UNSET)
        node_selector: PatchResourcePoolRequestNodeSelector | Unset
        if isinstance(_node_selector, Unset):
            node_selector = UNSET
        else:
            node_selector = PatchResourcePoolRequestNodeSelector.from_dict(
                _node_selector
            )

        _tolerations = d.pop("tolerations", UNSET)
        tolerations: list[Corev1Toleration] | Unset = UNSET
        if _tolerations is not UNSET:
            tolerations = []
            for tolerations_item_data in _tolerations:
                tolerations_item = Corev1Toleration.from_dict(tolerations_item_data)

                tolerations.append(tolerations_item)

        patch_resource_pool_request = cls(
            annotations=annotations,
            description=description,
            labels=labels,
            node_selector=node_selector,
            tolerations=tolerations,
        )

        patch_resource_pool_request.additional_properties = d
        return patch_resource_pool_request

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
