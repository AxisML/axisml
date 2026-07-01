from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..models.pod_phase import PodPhase
from ..types import UNSET, Unset

T = TypeVar("T", bound="Pod")


@_attrs_define
class Pod:
    """
    Example:
        {'name': 'resnet-train-7-worker-0', 'nodeName': 'gpu-node-03', 'phase': 'Running', 'replicaIndex': 0,
            'restartCount': 0, 'role': 'worker', 'startedAt': '2026-06-28T09:00:00Z'}

    Attributes:
        name (str): Pod name.
        phase (PodPhase): Pod lifecycle phase (Pending, Running, Succeeded, Failed, Unknown).
        exit_code (int | None | Unset): Exit code of the container's main process.
        finished_at (datetime.datetime | None | Unset): Time the pod terminated.
        message (str | Unset): Human-readable status detail for the pod.
        node_name (str | Unset): Node the pod is scheduled on.
        replica_index (int | Unset): Zero-based replica index within the role.
        restart_count (int | Unset): Number of times the pod's container has restarted.
        role (str | Unset): Topology role the pod belongs to (e.g. master, worker).
        started_at (datetime.datetime | None | Unset): Time the pod started running.
    """

    name: str
    phase: PodPhase
    exit_code: int | None | Unset = UNSET
    finished_at: datetime.datetime | None | Unset = UNSET
    message: str | Unset = UNSET
    node_name: str | Unset = UNSET
    replica_index: int | Unset = UNSET
    restart_count: int | Unset = UNSET
    role: str | Unset = UNSET
    started_at: datetime.datetime | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        name = self.name

        phase = self.phase.value

        exit_code: int | None | Unset
        if isinstance(self.exit_code, Unset):
            exit_code = UNSET
        else:
            exit_code = self.exit_code

        finished_at: None | str | Unset
        if isinstance(self.finished_at, Unset):
            finished_at = UNSET
        elif isinstance(self.finished_at, datetime.datetime):
            finished_at = self.finished_at.isoformat()
        else:
            finished_at = self.finished_at

        message = self.message

        node_name = self.node_name

        replica_index = self.replica_index

        restart_count = self.restart_count

        role = self.role

        started_at: None | str | Unset
        if isinstance(self.started_at, Unset):
            started_at = UNSET
        elif isinstance(self.started_at, datetime.datetime):
            started_at = self.started_at.isoformat()
        else:
            started_at = self.started_at

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
                "phase": phase,
            }
        )
        if exit_code is not UNSET:
            field_dict["exitCode"] = exit_code
        if finished_at is not UNSET:
            field_dict["finishedAt"] = finished_at
        if message is not UNSET:
            field_dict["message"] = message
        if node_name is not UNSET:
            field_dict["nodeName"] = node_name
        if replica_index is not UNSET:
            field_dict["replicaIndex"] = replica_index
        if restart_count is not UNSET:
            field_dict["restartCount"] = restart_count
        if role is not UNSET:
            field_dict["role"] = role
        if started_at is not UNSET:
            field_dict["startedAt"] = started_at

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        phase = PodPhase(d.pop("phase"))

        def _parse_exit_code(data: object) -> int | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(int | None | Unset, data)

        exit_code = _parse_exit_code(d.pop("exitCode", UNSET))

        def _parse_finished_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                finished_at_type_0 = datetime.datetime.fromisoformat(data)

                return finished_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        finished_at = _parse_finished_at(d.pop("finishedAt", UNSET))

        message = d.pop("message", UNSET)

        node_name = d.pop("nodeName", UNSET)

        replica_index = d.pop("replicaIndex", UNSET)

        restart_count = d.pop("restartCount", UNSET)

        role = d.pop("role", UNSET)

        def _parse_started_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                started_at_type_0 = datetime.datetime.fromisoformat(data)

                return started_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        started_at = _parse_started_at(d.pop("startedAt", UNSET))

        pod = cls(
            name=name,
            phase=phase,
            exit_code=exit_code,
            finished_at=finished_at,
            message=message,
            node_name=node_name,
            replica_index=replica_index,
            restart_count=restart_count,
            role=role,
            started_at=started_at,
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
