from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.pod_labels import PodLabels


T = TypeVar("T", bound="Pod")


@_attrs_define
class Pod:
    """
    Example:
        {'labels': {'compute.axisml.io/role': 'worker', 'compute.axisml.io/run-id':
            'b7d9e3f1-1a2b-3c4d-5e6f-708192a3b4c5'}, 'name': 'resnet-train-7-worker-0', 'namespace': 'team-vision',
            'nodeName': 'gpu-node-a100-03', 'phase': 'Running'}

    Attributes:
        name (str): Pod name.
        namespace (str): Namespace the pod runs in.
        phase (str): Pod lifecycle phase (Pending, Running, Succeeded, Failed, Unknown).
        labels (PodLabels | Unset): Pod labels.
        node_name (str | Unset): Node the pod is scheduled onto.
    """

    name: str
    namespace: str
    phase: str
    labels: PodLabels | Unset = UNSET
    node_name: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        namespace = self.namespace

        phase = self.phase

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        node_name = self.node_name

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
                "namespace": namespace,
                "phase": phase,
            }
        )
        if labels is not UNSET:
            field_dict["labels"] = labels
        if node_name is not UNSET:
            field_dict["nodeName"] = node_name

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.pod_labels import PodLabels

        d = dict(src_dict)
        name = d.pop("name")

        namespace = d.pop("namespace")

        phase = d.pop("phase")

        _labels = d.pop("labels", UNSET)
        labels: PodLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = PodLabels.from_dict(_labels)

        node_name = d.pop("nodeName", UNSET)

        pod = cls(
            name=name,
            namespace=namespace,
            phase=phase,
            labels=labels,
            node_name=node_name,
        )

        pod.additional_properties = d
        return pod

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
