from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_env_from_source import Corev1EnvFromSource
    from ..models.corev_1_env_var import Corev1EnvVar
    from ..models.corev_1_resource_requirements import Corev1ResourceRequirements
    from ..models.corev_1_volume import Corev1Volume
    from ..models.corev_1_volume_mount import Corev1VolumeMount


T = TypeVar("T", bound="MLRunPodTemplateSubset")


@_attrs_define
class MLRunPodTemplateSubset:
    """
    Attributes:
        image (str):
        args (list[str] | Unset):
        command (list[str] | Unset):
        env (list[Corev1EnvVar] | Unset):
        env_from (list[Corev1EnvFromSource] | Unset):
        image_pull_policy (str | Unset):
        resources (Corev1ResourceRequirements | Unset):
        volume_mounts (list[Corev1VolumeMount] | Unset):
        volumes (list[Corev1Volume] | Unset):
        working_dir (str | Unset):
    """

    image: str
    args: list[str] | Unset = UNSET
    command: list[str] | Unset = UNSET
    env: list[Corev1EnvVar] | Unset = UNSET
    env_from: list[Corev1EnvFromSource] | Unset = UNSET
    image_pull_policy: str | Unset = UNSET
    resources: Corev1ResourceRequirements | Unset = UNSET
    volume_mounts: list[Corev1VolumeMount] | Unset = UNSET
    volumes: list[Corev1Volume] | Unset = UNSET
    working_dir: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        image = self.image

        args: list[str] | Unset = UNSET
        if not isinstance(self.args, Unset):
            args = self.args

        command: list[str] | Unset = UNSET
        if not isinstance(self.command, Unset):
            command = self.command

        env: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.env, Unset):
            env = []
            for env_item_data in self.env:
                env_item = env_item_data.to_dict()
                env.append(env_item)

        env_from: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.env_from, Unset):
            env_from = []
            for env_from_item_data in self.env_from:
                env_from_item = env_from_item_data.to_dict()
                env_from.append(env_from_item)

        image_pull_policy = self.image_pull_policy

        resources: dict[str, Any] | Unset = UNSET
        if not isinstance(self.resources, Unset):
            resources = self.resources.to_dict()

        volume_mounts: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.volume_mounts, Unset):
            volume_mounts = []
            for volume_mounts_item_data in self.volume_mounts:
                volume_mounts_item = volume_mounts_item_data.to_dict()
                volume_mounts.append(volume_mounts_item)

        volumes: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.volumes, Unset):
            volumes = []
            for volumes_item_data in self.volumes:
                volumes_item = volumes_item_data.to_dict()
                volumes.append(volumes_item)

        working_dir = self.working_dir

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "image": image,
            }
        )
        if args is not UNSET:
            field_dict["args"] = args
        if command is not UNSET:
            field_dict["command"] = command
        if env is not UNSET:
            field_dict["env"] = env
        if env_from is not UNSET:
            field_dict["envFrom"] = env_from
        if image_pull_policy is not UNSET:
            field_dict["imagePullPolicy"] = image_pull_policy
        if resources is not UNSET:
            field_dict["resources"] = resources
        if volume_mounts is not UNSET:
            field_dict["volumeMounts"] = volume_mounts
        if volumes is not UNSET:
            field_dict["volumes"] = volumes
        if working_dir is not UNSET:
            field_dict["workingDir"] = working_dir

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_env_from_source import Corev1EnvFromSource
        from ..models.corev_1_env_var import Corev1EnvVar
        from ..models.corev_1_resource_requirements import Corev1ResourceRequirements
        from ..models.corev_1_volume import Corev1Volume
        from ..models.corev_1_volume_mount import Corev1VolumeMount

        d = dict(src_dict)
        image = d.pop("image")

        args = cast(list[str], d.pop("args", UNSET))

        command = cast(list[str], d.pop("command", UNSET))

        _env = d.pop("env", UNSET)
        env: list[Corev1EnvVar] | Unset = UNSET
        if _env is not UNSET:
            env = []
            for env_item_data in _env:
                env_item = Corev1EnvVar.from_dict(env_item_data)

                env.append(env_item)

        _env_from = d.pop("envFrom", UNSET)
        env_from: list[Corev1EnvFromSource] | Unset = UNSET
        if _env_from is not UNSET:
            env_from = []
            for env_from_item_data in _env_from:
                env_from_item = Corev1EnvFromSource.from_dict(env_from_item_data)

                env_from.append(env_from_item)

        image_pull_policy = d.pop("imagePullPolicy", UNSET)

        _resources = d.pop("resources", UNSET)
        resources: Corev1ResourceRequirements | Unset
        if isinstance(_resources, Unset):
            resources = UNSET
        else:
            resources = Corev1ResourceRequirements.from_dict(_resources)

        _volume_mounts = d.pop("volumeMounts", UNSET)
        volume_mounts: list[Corev1VolumeMount] | Unset = UNSET
        if _volume_mounts is not UNSET:
            volume_mounts = []
            for volume_mounts_item_data in _volume_mounts:
                volume_mounts_item = Corev1VolumeMount.from_dict(
                    volume_mounts_item_data
                )

                volume_mounts.append(volume_mounts_item)

        _volumes = d.pop("volumes", UNSET)
        volumes: list[Corev1Volume] | Unset = UNSET
        if _volumes is not UNSET:
            volumes = []
            for volumes_item_data in _volumes:
                volumes_item = Corev1Volume.from_dict(volumes_item_data)

                volumes.append(volumes_item)

        working_dir = d.pop("workingDir", UNSET)

        ml_run_pod_template_subset = cls(
            image=image,
            args=args,
            command=command,
            env=env,
            env_from=env_from,
            image_pull_policy=image_pull_policy,
            resources=resources,
            volume_mounts=volume_mounts,
            volumes=volumes,
            working_dir=working_dir,
        )

        ml_run_pod_template_subset.additional_properties = d
        return ml_run_pod_template_subset

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
