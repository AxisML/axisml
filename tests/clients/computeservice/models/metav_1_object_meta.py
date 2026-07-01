from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.metav_1_managed_fields_entry import Metav1ManagedFieldsEntry
    from ..models.metav_1_object_meta_annotations import Metav1ObjectMetaAnnotations
    from ..models.metav_1_object_meta_labels import Metav1ObjectMetaLabels
    from ..models.metav_1_owner_reference import Metav1OwnerReference
    from ..models.metav_1_time import Metav1Time


T = TypeVar("T", bound="Metav1ObjectMeta")


@_attrs_define
class Metav1ObjectMeta:
    """
    Attributes:
        annotations (Metav1ObjectMetaAnnotations | Unset):
        creation_timestamp (Metav1Time | Unset):
        deletion_grace_period_seconds (int | None | Unset):
        deletion_timestamp (Metav1Time | None | Unset):
        finalizers (list[str] | Unset):
        generate_name (str | Unset):
        generation (int | Unset):
        labels (Metav1ObjectMetaLabels | Unset):
        managed_fields (list[Metav1ManagedFieldsEntry] | Unset):
        name (str | Unset):
        namespace (str | Unset):
        owner_references (list[Metav1OwnerReference] | Unset):
        resource_version (str | Unset):
        self_link (str | Unset):
        uid (str | Unset):
    """

    annotations: Metav1ObjectMetaAnnotations | Unset = UNSET
    creation_timestamp: Metav1Time | Unset = UNSET
    deletion_grace_period_seconds: int | None | Unset = UNSET
    deletion_timestamp: Metav1Time | None | Unset = UNSET
    finalizers: list[str] | Unset = UNSET
    generate_name: str | Unset = UNSET
    generation: int | Unset = UNSET
    labels: Metav1ObjectMetaLabels | Unset = UNSET
    managed_fields: list[Metav1ManagedFieldsEntry] | Unset = UNSET
    name: str | Unset = UNSET
    namespace: str | Unset = UNSET
    owner_references: list[Metav1OwnerReference] | Unset = UNSET
    resource_version: str | Unset = UNSET
    self_link: str | Unset = UNSET
    uid: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.metav_1_time import Metav1Time

        annotations: dict[str, Any] | Unset = UNSET
        if not isinstance(self.annotations, Unset):
            annotations = self.annotations.to_dict()

        creation_timestamp: dict[str, Any] | Unset = UNSET
        if not isinstance(self.creation_timestamp, Unset):
            creation_timestamp = self.creation_timestamp.to_dict()

        deletion_grace_period_seconds: int | None | Unset
        if isinstance(self.deletion_grace_period_seconds, Unset):
            deletion_grace_period_seconds = UNSET
        else:
            deletion_grace_period_seconds = self.deletion_grace_period_seconds

        deletion_timestamp: dict[str, Any] | None | Unset
        if isinstance(self.deletion_timestamp, Unset):
            deletion_timestamp = UNSET
        elif isinstance(self.deletion_timestamp, Metav1Time):
            deletion_timestamp = self.deletion_timestamp.to_dict()
        else:
            deletion_timestamp = self.deletion_timestamp

        finalizers: list[str] | Unset = UNSET
        if not isinstance(self.finalizers, Unset):
            finalizers = self.finalizers

        generate_name = self.generate_name

        generation = self.generation

        labels: dict[str, Any] | Unset = UNSET
        if not isinstance(self.labels, Unset):
            labels = self.labels.to_dict()

        managed_fields: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.managed_fields, Unset):
            managed_fields = []
            for managed_fields_item_data in self.managed_fields:
                managed_fields_item = managed_fields_item_data.to_dict()
                managed_fields.append(managed_fields_item)

        name = self.name

        namespace = self.namespace

        owner_references: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.owner_references, Unset):
            owner_references = []
            for owner_references_item_data in self.owner_references:
                owner_references_item = owner_references_item_data.to_dict()
                owner_references.append(owner_references_item)

        resource_version = self.resource_version

        self_link = self.self_link

        uid = self.uid

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if annotations is not UNSET:
            field_dict["annotations"] = annotations
        if creation_timestamp is not UNSET:
            field_dict["creationTimestamp"] = creation_timestamp
        if deletion_grace_period_seconds is not UNSET:
            field_dict["deletionGracePeriodSeconds"] = deletion_grace_period_seconds
        if deletion_timestamp is not UNSET:
            field_dict["deletionTimestamp"] = deletion_timestamp
        if finalizers is not UNSET:
            field_dict["finalizers"] = finalizers
        if generate_name is not UNSET:
            field_dict["generateName"] = generate_name
        if generation is not UNSET:
            field_dict["generation"] = generation
        if labels is not UNSET:
            field_dict["labels"] = labels
        if managed_fields is not UNSET:
            field_dict["managedFields"] = managed_fields
        if name is not UNSET:
            field_dict["name"] = name
        if namespace is not UNSET:
            field_dict["namespace"] = namespace
        if owner_references is not UNSET:
            field_dict["ownerReferences"] = owner_references
        if resource_version is not UNSET:
            field_dict["resourceVersion"] = resource_version
        if self_link is not UNSET:
            field_dict["selfLink"] = self_link
        if uid is not UNSET:
            field_dict["uid"] = uid

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.metav_1_managed_fields_entry import Metav1ManagedFieldsEntry
        from ..models.metav_1_object_meta_annotations import Metav1ObjectMetaAnnotations
        from ..models.metav_1_object_meta_labels import Metav1ObjectMetaLabels
        from ..models.metav_1_owner_reference import Metav1OwnerReference
        from ..models.metav_1_time import Metav1Time

        d = dict(src_dict)
        _annotations = d.pop("annotations", UNSET)
        annotations: Metav1ObjectMetaAnnotations | Unset
        if isinstance(_annotations, Unset):
            annotations = UNSET
        else:
            annotations = Metav1ObjectMetaAnnotations.from_dict(_annotations)

        _creation_timestamp = d.pop("creationTimestamp", UNSET)
        creation_timestamp: Metav1Time | Unset
        if isinstance(_creation_timestamp, Unset):
            creation_timestamp = UNSET
        else:
            creation_timestamp = Metav1Time.from_dict(_creation_timestamp)

        def _parse_deletion_grace_period_seconds(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        deletion_grace_period_seconds = _parse_deletion_grace_period_seconds(
            d.pop("deletionGracePeriodSeconds", UNSET)
        )

        def _parse_deletion_timestamp(data: object) -> Metav1Time | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                deletion_timestamp_type_1 = Metav1Time.from_dict(data)

                return deletion_timestamp_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Metav1Time | None | Unset, data)

        deletion_timestamp = _parse_deletion_timestamp(
            d.pop("deletionTimestamp", UNSET)
        )

        finalizers = cast(list[str], d.pop("finalizers", UNSET))

        generate_name = d.pop("generateName", UNSET)

        generation = d.pop("generation", UNSET)

        _labels = d.pop("labels", UNSET)
        labels: Metav1ObjectMetaLabels | Unset
        if isinstance(_labels, Unset):
            labels = UNSET
        else:
            labels = Metav1ObjectMetaLabels.from_dict(_labels)

        _managed_fields = d.pop("managedFields", UNSET)
        managed_fields: list[Metav1ManagedFieldsEntry] | Unset = UNSET
        if _managed_fields is not UNSET:
            managed_fields = []
            for managed_fields_item_data in _managed_fields:
                managed_fields_item = Metav1ManagedFieldsEntry.from_dict(
                    managed_fields_item_data
                )

                managed_fields.append(managed_fields_item)

        name = d.pop("name", UNSET)

        namespace = d.pop("namespace", UNSET)

        _owner_references = d.pop("ownerReferences", UNSET)
        owner_references: list[Metav1OwnerReference] | Unset = UNSET
        if _owner_references is not UNSET:
            owner_references = []
            for owner_references_item_data in _owner_references:
                owner_references_item = Metav1OwnerReference.from_dict(
                    owner_references_item_data
                )

                owner_references.append(owner_references_item)

        resource_version = d.pop("resourceVersion", UNSET)

        self_link = d.pop("selfLink", UNSET)

        uid = d.pop("uid", UNSET)

        metav_1_object_meta = cls(
            annotations=annotations,
            creation_timestamp=creation_timestamp,
            deletion_grace_period_seconds=deletion_grace_period_seconds,
            deletion_timestamp=deletion_timestamp,
            finalizers=finalizers,
            generate_name=generate_name,
            generation=generation,
            labels=labels,
            managed_fields=managed_fields,
            name=name,
            namespace=namespace,
            owner_references=owner_references,
            resource_version=resource_version,
            self_link=self_link,
            uid=uid,
        )

        metav_1_object_meta.additional_properties = d
        return metav_1_object_meta

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
