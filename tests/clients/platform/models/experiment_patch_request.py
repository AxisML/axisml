from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.job_spec import JobSpec
    from ..models.string_map import StringMap


T = TypeVar("T", bound="ExperimentPatchRequest")


@_attrs_define
class ExperimentPatchRequest:
    """
    Example:
        {'description': 'Updated description.', 'displayName': 'BERT fine-tuning experiment (v2)'}

    Attributes:
        annotations (StringMap | Unset):
        description (str | Unset): Updated free-text experiment description.
        display_name (str | Unset): Updated human-readable experiment label.
        labels (StringMap | Unset):
        spec (JobSpec | None | Unset): Replacement run template (affects only Runs triggered afterwards); omit to patch
            metadata only.
    """

    annotations: StringMap | Unset = UNSET
    description: str | Unset = UNSET
    display_name: str | Unset = UNSET
    labels: StringMap | Unset = UNSET
    spec: JobSpec | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.job_spec import JobSpec

        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        description = self.description

        display_name = self.display_name

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        spec: dict[str, Any] | None | Unset
        if isinstance(self.spec, Unset):
            spec = UNSET
        elif isinstance(self.spec, JobSpec):
            spec = self.spec.to_dict()
        else:
            spec = self.spec

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if description is not UNSET:
            field_dict["description"] = description
        if display_name is not UNSET:
            field_dict["displayName"] = display_name
        if labels is not UNSET:
            field_dict["labels"] = labels
        if spec is not UNSET:
            field_dict["spec"] = spec

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.job_spec import JobSpec
        from ..models.string_map import StringMap

        d = dict(src_dict)
        _annotations = d.pop("annotations", UNSET)
        annotations: StringMap | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = StringMap.from_dict(_annotations)

        description = d.pop("description", UNSET)

        display_name = d.pop("displayName", UNSET)

        _labels = d.pop("labels", UNSET)
        labels: StringMap | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = StringMap.from_dict(_labels)

        def _parse_spec(data: object) -> JobSpec | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                spec_type_1 = JobSpec.from_dict(data)

                return spec_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(JobSpec | None | Unset, data)

        spec = _parse_spec(d.pop("spec", UNSET))

        experiment_patch_request = cls(
            annotations=annotations,
            description=description,
            display_name=display_name,
            labels=labels,
            spec=spec,
        )

        experiment_patch_request.additional_properties = d
        return experiment_patch_request

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
