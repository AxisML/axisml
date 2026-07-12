from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.corev_1_azure_disk_volume_source import Corev1AzureDiskVolumeSource
    from ..models.corev_1_azure_file_volume_source import Corev1AzureFileVolumeSource
    from ..models.corev_1_ceph_fs_volume_source import Corev1CephFSVolumeSource
    from ..models.corev_1_cinder_volume_source import Corev1CinderVolumeSource
    from ..models.corev_1_config_map_volume_source import Corev1ConfigMapVolumeSource
    from ..models.corev_1_downward_api_volume_source import (
        Corev1DownwardAPIVolumeSource,
    )
    from ..models.corev_1_empty_dir_volume_source import Corev1EmptyDirVolumeSource
    from ..models.corev_1_ephemeral_volume_source import Corev1EphemeralVolumeSource
    from ..models.corev_1_flex_volume_source import Corev1FlexVolumeSource
    from ..models.corev_1_flocker_volume_source import Corev1FlockerVolumeSource
    from ..models.corev_1_git_repo_volume_source import Corev1GitRepoVolumeSource
    from ..models.corev_1_glusterfs_volume_source import Corev1GlusterfsVolumeSource
    from ..models.corev_1_host_path_volume_source import Corev1HostPathVolumeSource
    from ..models.corev_1_image_volume_source import Corev1ImageVolumeSource
    from ..models.corev_1_persistent_volume_claim_volume_source import (
        Corev1PersistentVolumeClaimVolumeSource,
    )
    from ..models.corev_1_photon_persistent_disk_volume_source import (
        Corev1PhotonPersistentDiskVolumeSource,
    )
    from ..models.corev_1_portworx_volume_source import Corev1PortworxVolumeSource
    from ..models.corev_1_projected_volume_source import Corev1ProjectedVolumeSource
    from ..models.corev_1_quobyte_volume_source import Corev1QuobyteVolumeSource
    from ..models.corev_1_scale_io_volume_source import Corev1ScaleIOVolumeSource
    from ..models.corev_1_secret_volume_source import Corev1SecretVolumeSource
    from ..models.corev_1_storage_os_volume_source import Corev1StorageOSVolumeSource
    from ..models.corev_1_vsphere_virtual_disk_volume_source import (
        Corev1VsphereVirtualDiskVolumeSource,
    )
    from ..models.corev_1aws_elastic_block_store_volume_source import (
        Corev1AWSElasticBlockStoreVolumeSource,
    )
    from ..models.corev_1csi_volume_source import Corev1CSIVolumeSource
    from ..models.corev_1fc_volume_source import Corev1FCVolumeSource
    from ..models.corev_1gce_persistent_disk_volume_source import (
        Corev1GCEPersistentDiskVolumeSource,
    )
    from ..models.corev_1iscsi_volume_source import Corev1ISCSIVolumeSource
    from ..models.corev_1nfs_volume_source import Corev1NFSVolumeSource
    from ..models.corev_1rbd_volume_source import Corev1RBDVolumeSource


T = TypeVar("T", bound="Corev1Volume")


@_attrs_define
class Corev1Volume:
    """
    Attributes:
        name (str):
        aws_elastic_block_store (Corev1AWSElasticBlockStoreVolumeSource | None | Unset):
        azure_disk (Corev1AzureDiskVolumeSource | None | Unset):
        azure_file (Corev1AzureFileVolumeSource | None | Unset):
        cephfs (Corev1CephFSVolumeSource | None | Unset):
        cinder (Corev1CinderVolumeSource | None | Unset):
        config_map (Corev1ConfigMapVolumeSource | None | Unset):
        csi (Corev1CSIVolumeSource | None | Unset):
        downward_api (Corev1DownwardAPIVolumeSource | None | Unset):
        empty_dir (Corev1EmptyDirVolumeSource | None | Unset):
        ephemeral (Corev1EphemeralVolumeSource | None | Unset):
        fc (Corev1FCVolumeSource | None | Unset):
        flex_volume (Corev1FlexVolumeSource | None | Unset):
        flocker (Corev1FlockerVolumeSource | None | Unset):
        gce_persistent_disk (Corev1GCEPersistentDiskVolumeSource | None | Unset):
        git_repo (Corev1GitRepoVolumeSource | None | Unset):
        glusterfs (Corev1GlusterfsVolumeSource | None | Unset):
        host_path (Corev1HostPathVolumeSource | None | Unset):
        image (Corev1ImageVolumeSource | None | Unset):
        iscsi (Corev1ISCSIVolumeSource | None | Unset):
        nfs (Corev1NFSVolumeSource | None | Unset):
        persistent_volume_claim (Corev1PersistentVolumeClaimVolumeSource | None | Unset):
        photon_persistent_disk (Corev1PhotonPersistentDiskVolumeSource | None | Unset):
        portworx_volume (Corev1PortworxVolumeSource | None | Unset):
        projected (Corev1ProjectedVolumeSource | None | Unset):
        quobyte (Corev1QuobyteVolumeSource | None | Unset):
        rbd (Corev1RBDVolumeSource | None | Unset):
        scale_io (Corev1ScaleIOVolumeSource | None | Unset):
        secret (Corev1SecretVolumeSource | None | Unset):
        storageos (Corev1StorageOSVolumeSource | None | Unset):
        vsphere_volume (Corev1VsphereVirtualDiskVolumeSource | None | Unset):
    """

    name: str
    aws_elastic_block_store: Corev1AWSElasticBlockStoreVolumeSource | None | Unset = (
        UNSET
    )
    azure_disk: Corev1AzureDiskVolumeSource | None | Unset = UNSET
    azure_file: Corev1AzureFileVolumeSource | None | Unset = UNSET
    cephfs: Corev1CephFSVolumeSource | None | Unset = UNSET
    cinder: Corev1CinderVolumeSource | None | Unset = UNSET
    config_map: Corev1ConfigMapVolumeSource | None | Unset = UNSET
    csi: Corev1CSIVolumeSource | None | Unset = UNSET
    downward_api: Corev1DownwardAPIVolumeSource | None | Unset = UNSET
    empty_dir: Corev1EmptyDirVolumeSource | None | Unset = UNSET
    ephemeral: Corev1EphemeralVolumeSource | None | Unset = UNSET
    fc: Corev1FCVolumeSource | None | Unset = UNSET
    flex_volume: Corev1FlexVolumeSource | None | Unset = UNSET
    flocker: Corev1FlockerVolumeSource | None | Unset = UNSET
    gce_persistent_disk: Corev1GCEPersistentDiskVolumeSource | None | Unset = UNSET
    git_repo: Corev1GitRepoVolumeSource | None | Unset = UNSET
    glusterfs: Corev1GlusterfsVolumeSource | None | Unset = UNSET
    host_path: Corev1HostPathVolumeSource | None | Unset = UNSET
    image: Corev1ImageVolumeSource | None | Unset = UNSET
    iscsi: Corev1ISCSIVolumeSource | None | Unset = UNSET
    nfs: Corev1NFSVolumeSource | None | Unset = UNSET
    persistent_volume_claim: Corev1PersistentVolumeClaimVolumeSource | None | Unset = (
        UNSET
    )
    photon_persistent_disk: Corev1PhotonPersistentDiskVolumeSource | None | Unset = (
        UNSET
    )
    portworx_volume: Corev1PortworxVolumeSource | None | Unset = UNSET
    projected: Corev1ProjectedVolumeSource | None | Unset = UNSET
    quobyte: Corev1QuobyteVolumeSource | None | Unset = UNSET
    rbd: Corev1RBDVolumeSource | None | Unset = UNSET
    scale_io: Corev1ScaleIOVolumeSource | None | Unset = UNSET
    secret: Corev1SecretVolumeSource | None | Unset = UNSET
    storageos: Corev1StorageOSVolumeSource | None | Unset = UNSET
    vsphere_volume: Corev1VsphereVirtualDiskVolumeSource | None | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.corev_1_azure_disk_volume_source import (
            Corev1AzureDiskVolumeSource,
        )
        from ..models.corev_1_azure_file_volume_source import (
            Corev1AzureFileVolumeSource,
        )
        from ..models.corev_1_ceph_fs_volume_source import Corev1CephFSVolumeSource
        from ..models.corev_1_cinder_volume_source import Corev1CinderVolumeSource
        from ..models.corev_1_config_map_volume_source import (
            Corev1ConfigMapVolumeSource,
        )
        from ..models.corev_1_downward_api_volume_source import (
            Corev1DownwardAPIVolumeSource,
        )
        from ..models.corev_1_empty_dir_volume_source import Corev1EmptyDirVolumeSource
        from ..models.corev_1_ephemeral_volume_source import Corev1EphemeralVolumeSource
        from ..models.corev_1_flex_volume_source import Corev1FlexVolumeSource
        from ..models.corev_1_flocker_volume_source import Corev1FlockerVolumeSource
        from ..models.corev_1_git_repo_volume_source import Corev1GitRepoVolumeSource
        from ..models.corev_1_glusterfs_volume_source import Corev1GlusterfsVolumeSource
        from ..models.corev_1_host_path_volume_source import Corev1HostPathVolumeSource
        from ..models.corev_1_image_volume_source import Corev1ImageVolumeSource
        from ..models.corev_1_persistent_volume_claim_volume_source import (
            Corev1PersistentVolumeClaimVolumeSource,
        )
        from ..models.corev_1_photon_persistent_disk_volume_source import (
            Corev1PhotonPersistentDiskVolumeSource,
        )
        from ..models.corev_1_portworx_volume_source import Corev1PortworxVolumeSource
        from ..models.corev_1_projected_volume_source import Corev1ProjectedVolumeSource
        from ..models.corev_1_quobyte_volume_source import Corev1QuobyteVolumeSource
        from ..models.corev_1_scale_io_volume_source import Corev1ScaleIOVolumeSource
        from ..models.corev_1_secret_volume_source import Corev1SecretVolumeSource
        from ..models.corev_1_storage_os_volume_source import (
            Corev1StorageOSVolumeSource,
        )
        from ..models.corev_1_vsphere_virtual_disk_volume_source import (
            Corev1VsphereVirtualDiskVolumeSource,
        )
        from ..models.corev_1aws_elastic_block_store_volume_source import (
            Corev1AWSElasticBlockStoreVolumeSource,
        )
        from ..models.corev_1csi_volume_source import Corev1CSIVolumeSource
        from ..models.corev_1fc_volume_source import Corev1FCVolumeSource
        from ..models.corev_1gce_persistent_disk_volume_source import (
            Corev1GCEPersistentDiskVolumeSource,
        )
        from ..models.corev_1iscsi_volume_source import Corev1ISCSIVolumeSource
        from ..models.corev_1nfs_volume_source import Corev1NFSVolumeSource
        from ..models.corev_1rbd_volume_source import Corev1RBDVolumeSource

        name = self.name

        aws_elastic_block_store: dict[str, Any] | None | Unset
        if isinstance(self.aws_elastic_block_store, Unset):
            aws_elastic_block_store = UNSET
        elif isinstance(
            self.aws_elastic_block_store, Corev1AWSElasticBlockStoreVolumeSource
        ):
            aws_elastic_block_store = self.aws_elastic_block_store.to_dict()
        else:
            aws_elastic_block_store = self.aws_elastic_block_store

        azure_disk: dict[str, Any] | None | Unset
        if isinstance(self.azure_disk, Unset):
            azure_disk = UNSET
        elif isinstance(self.azure_disk, Corev1AzureDiskVolumeSource):
            azure_disk = self.azure_disk.to_dict()
        else:
            azure_disk = self.azure_disk

        azure_file: dict[str, Any] | None | Unset
        if isinstance(self.azure_file, Unset):
            azure_file = UNSET
        elif isinstance(self.azure_file, Corev1AzureFileVolumeSource):
            azure_file = self.azure_file.to_dict()
        else:
            azure_file = self.azure_file

        cephfs: dict[str, Any] | None | Unset
        if isinstance(self.cephfs, Unset):
            cephfs = UNSET
        elif isinstance(self.cephfs, Corev1CephFSVolumeSource):
            cephfs = self.cephfs.to_dict()
        else:
            cephfs = self.cephfs

        cinder: dict[str, Any] | None | Unset
        if isinstance(self.cinder, Unset):
            cinder = UNSET
        elif isinstance(self.cinder, Corev1CinderVolumeSource):
            cinder = self.cinder.to_dict()
        else:
            cinder = self.cinder

        config_map: dict[str, Any] | None | Unset
        if isinstance(self.config_map, Unset):
            config_map = UNSET
        elif isinstance(self.config_map, Corev1ConfigMapVolumeSource):
            config_map = self.config_map.to_dict()
        else:
            config_map = self.config_map

        csi: dict[str, Any] | None | Unset
        if isinstance(self.csi, Unset):
            csi = UNSET
        elif isinstance(self.csi, Corev1CSIVolumeSource):
            csi = self.csi.to_dict()
        else:
            csi = self.csi

        downward_api: dict[str, Any] | None | Unset
        if isinstance(self.downward_api, Unset):
            downward_api = UNSET
        elif isinstance(self.downward_api, Corev1DownwardAPIVolumeSource):
            downward_api = self.downward_api.to_dict()
        else:
            downward_api = self.downward_api

        empty_dir: dict[str, Any] | None | Unset
        if isinstance(self.empty_dir, Unset):
            empty_dir = UNSET
        elif isinstance(self.empty_dir, Corev1EmptyDirVolumeSource):
            empty_dir = self.empty_dir.to_dict()
        else:
            empty_dir = self.empty_dir

        ephemeral: dict[str, Any] | None | Unset
        if isinstance(self.ephemeral, Unset):
            ephemeral = UNSET
        elif isinstance(self.ephemeral, Corev1EphemeralVolumeSource):
            ephemeral = self.ephemeral.to_dict()
        else:
            ephemeral = self.ephemeral

        fc: dict[str, Any] | None | Unset
        if isinstance(self.fc, Unset):
            fc = UNSET
        elif isinstance(self.fc, Corev1FCVolumeSource):
            fc = self.fc.to_dict()
        else:
            fc = self.fc

        flex_volume: dict[str, Any] | None | Unset
        if isinstance(self.flex_volume, Unset):
            flex_volume = UNSET
        elif isinstance(self.flex_volume, Corev1FlexVolumeSource):
            flex_volume = self.flex_volume.to_dict()
        else:
            flex_volume = self.flex_volume

        flocker: dict[str, Any] | None | Unset
        if isinstance(self.flocker, Unset):
            flocker = UNSET
        elif isinstance(self.flocker, Corev1FlockerVolumeSource):
            flocker = self.flocker.to_dict()
        else:
            flocker = self.flocker

        gce_persistent_disk: dict[str, Any] | None | Unset
        if isinstance(self.gce_persistent_disk, Unset):
            gce_persistent_disk = UNSET
        elif isinstance(self.gce_persistent_disk, Corev1GCEPersistentDiskVolumeSource):
            gce_persistent_disk = self.gce_persistent_disk.to_dict()
        else:
            gce_persistent_disk = self.gce_persistent_disk

        git_repo: dict[str, Any] | None | Unset
        if isinstance(self.git_repo, Unset):
            git_repo = UNSET
        elif isinstance(self.git_repo, Corev1GitRepoVolumeSource):
            git_repo = self.git_repo.to_dict()
        else:
            git_repo = self.git_repo

        glusterfs: dict[str, Any] | None | Unset
        if isinstance(self.glusterfs, Unset):
            glusterfs = UNSET
        elif isinstance(self.glusterfs, Corev1GlusterfsVolumeSource):
            glusterfs = self.glusterfs.to_dict()
        else:
            glusterfs = self.glusterfs

        host_path: dict[str, Any] | None | Unset
        if isinstance(self.host_path, Unset):
            host_path = UNSET
        elif isinstance(self.host_path, Corev1HostPathVolumeSource):
            host_path = self.host_path.to_dict()
        else:
            host_path = self.host_path

        image: dict[str, Any] | None | Unset
        if isinstance(self.image, Unset):
            image = UNSET
        elif isinstance(self.image, Corev1ImageVolumeSource):
            image = self.image.to_dict()
        else:
            image = self.image

        iscsi: dict[str, Any] | None | Unset
        if isinstance(self.iscsi, Unset):
            iscsi = UNSET
        elif isinstance(self.iscsi, Corev1ISCSIVolumeSource):
            iscsi = self.iscsi.to_dict()
        else:
            iscsi = self.iscsi

        nfs: dict[str, Any] | None | Unset
        if isinstance(self.nfs, Unset):
            nfs = UNSET
        elif isinstance(self.nfs, Corev1NFSVolumeSource):
            nfs = self.nfs.to_dict()
        else:
            nfs = self.nfs

        persistent_volume_claim: dict[str, Any] | None | Unset
        if isinstance(self.persistent_volume_claim, Unset):
            persistent_volume_claim = UNSET
        elif isinstance(
            self.persistent_volume_claim, Corev1PersistentVolumeClaimVolumeSource
        ):
            persistent_volume_claim = self.persistent_volume_claim.to_dict()
        else:
            persistent_volume_claim = self.persistent_volume_claim

        photon_persistent_disk: dict[str, Any] | None | Unset
        if isinstance(self.photon_persistent_disk, Unset):
            photon_persistent_disk = UNSET
        elif isinstance(
            self.photon_persistent_disk, Corev1PhotonPersistentDiskVolumeSource
        ):
            photon_persistent_disk = self.photon_persistent_disk.to_dict()
        else:
            photon_persistent_disk = self.photon_persistent_disk

        portworx_volume: dict[str, Any] | None | Unset
        if isinstance(self.portworx_volume, Unset):
            portworx_volume = UNSET
        elif isinstance(self.portworx_volume, Corev1PortworxVolumeSource):
            portworx_volume = self.portworx_volume.to_dict()
        else:
            portworx_volume = self.portworx_volume

        projected: dict[str, Any] | None | Unset
        if isinstance(self.projected, Unset):
            projected = UNSET
        elif isinstance(self.projected, Corev1ProjectedVolumeSource):
            projected = self.projected.to_dict()
        else:
            projected = self.projected

        quobyte: dict[str, Any] | None | Unset
        if isinstance(self.quobyte, Unset):
            quobyte = UNSET
        elif isinstance(self.quobyte, Corev1QuobyteVolumeSource):
            quobyte = self.quobyte.to_dict()
        else:
            quobyte = self.quobyte

        rbd: dict[str, Any] | None | Unset
        if isinstance(self.rbd, Unset):
            rbd = UNSET
        elif isinstance(self.rbd, Corev1RBDVolumeSource):
            rbd = self.rbd.to_dict()
        else:
            rbd = self.rbd

        scale_io: dict[str, Any] | None | Unset
        if isinstance(self.scale_io, Unset):
            scale_io = UNSET
        elif isinstance(self.scale_io, Corev1ScaleIOVolumeSource):
            scale_io = self.scale_io.to_dict()
        else:
            scale_io = self.scale_io

        secret: dict[str, Any] | None | Unset
        if isinstance(self.secret, Unset):
            secret = UNSET
        elif isinstance(self.secret, Corev1SecretVolumeSource):
            secret = self.secret.to_dict()
        else:
            secret = self.secret

        storageos: dict[str, Any] | None | Unset
        if isinstance(self.storageos, Unset):
            storageos = UNSET
        elif isinstance(self.storageos, Corev1StorageOSVolumeSource):
            storageos = self.storageos.to_dict()
        else:
            storageos = self.storageos

        vsphere_volume: dict[str, Any] | None | Unset
        if isinstance(self.vsphere_volume, Unset):
            vsphere_volume = UNSET
        elif isinstance(self.vsphere_volume, Corev1VsphereVirtualDiskVolumeSource):
            vsphere_volume = self.vsphere_volume.to_dict()
        else:
            vsphere_volume = self.vsphere_volume

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "name": name,
            }
        )
        if aws_elastic_block_store is not UNSET:
            field_dict["awsElasticBlockStore"] = aws_elastic_block_store
        if azure_disk is not UNSET:
            field_dict["azureDisk"] = azure_disk
        if azure_file is not UNSET:
            field_dict["azureFile"] = azure_file
        if cephfs is not UNSET:
            field_dict["cephfs"] = cephfs
        if cinder is not UNSET:
            field_dict["cinder"] = cinder
        if config_map is not UNSET:
            field_dict["configMap"] = config_map
        if csi is not UNSET:
            field_dict["csi"] = csi
        if downward_api is not UNSET:
            field_dict["downwardAPI"] = downward_api
        if empty_dir is not UNSET:
            field_dict["emptyDir"] = empty_dir
        if ephemeral is not UNSET:
            field_dict["ephemeral"] = ephemeral
        if fc is not UNSET:
            field_dict["fc"] = fc
        if flex_volume is not UNSET:
            field_dict["flexVolume"] = flex_volume
        if flocker is not UNSET:
            field_dict["flocker"] = flocker
        if gce_persistent_disk is not UNSET:
            field_dict["gcePersistentDisk"] = gce_persistent_disk
        if git_repo is not UNSET:
            field_dict["gitRepo"] = git_repo
        if glusterfs is not UNSET:
            field_dict["glusterfs"] = glusterfs
        if host_path is not UNSET:
            field_dict["hostPath"] = host_path
        if image is not UNSET:
            field_dict["image"] = image
        if iscsi is not UNSET:
            field_dict["iscsi"] = iscsi
        if nfs is not UNSET:
            field_dict["nfs"] = nfs
        if persistent_volume_claim is not UNSET:
            field_dict["persistentVolumeClaim"] = persistent_volume_claim
        if photon_persistent_disk is not UNSET:
            field_dict["photonPersistentDisk"] = photon_persistent_disk
        if portworx_volume is not UNSET:
            field_dict["portworxVolume"] = portworx_volume
        if projected is not UNSET:
            field_dict["projected"] = projected
        if quobyte is not UNSET:
            field_dict["quobyte"] = quobyte
        if rbd is not UNSET:
            field_dict["rbd"] = rbd
        if scale_io is not UNSET:
            field_dict["scaleIO"] = scale_io
        if secret is not UNSET:
            field_dict["secret"] = secret
        if storageos is not UNSET:
            field_dict["storageos"] = storageos
        if vsphere_volume is not UNSET:
            field_dict["vsphereVolume"] = vsphere_volume

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.corev_1_azure_disk_volume_source import (
            Corev1AzureDiskVolumeSource,
        )
        from ..models.corev_1_azure_file_volume_source import (
            Corev1AzureFileVolumeSource,
        )
        from ..models.corev_1_ceph_fs_volume_source import Corev1CephFSVolumeSource
        from ..models.corev_1_cinder_volume_source import Corev1CinderVolumeSource
        from ..models.corev_1_config_map_volume_source import (
            Corev1ConfigMapVolumeSource,
        )
        from ..models.corev_1_downward_api_volume_source import (
            Corev1DownwardAPIVolumeSource,
        )
        from ..models.corev_1_empty_dir_volume_source import Corev1EmptyDirVolumeSource
        from ..models.corev_1_ephemeral_volume_source import Corev1EphemeralVolumeSource
        from ..models.corev_1_flex_volume_source import Corev1FlexVolumeSource
        from ..models.corev_1_flocker_volume_source import Corev1FlockerVolumeSource
        from ..models.corev_1_git_repo_volume_source import Corev1GitRepoVolumeSource
        from ..models.corev_1_glusterfs_volume_source import Corev1GlusterfsVolumeSource
        from ..models.corev_1_host_path_volume_source import Corev1HostPathVolumeSource
        from ..models.corev_1_image_volume_source import Corev1ImageVolumeSource
        from ..models.corev_1_persistent_volume_claim_volume_source import (
            Corev1PersistentVolumeClaimVolumeSource,
        )
        from ..models.corev_1_photon_persistent_disk_volume_source import (
            Corev1PhotonPersistentDiskVolumeSource,
        )
        from ..models.corev_1_portworx_volume_source import Corev1PortworxVolumeSource
        from ..models.corev_1_projected_volume_source import Corev1ProjectedVolumeSource
        from ..models.corev_1_quobyte_volume_source import Corev1QuobyteVolumeSource
        from ..models.corev_1_scale_io_volume_source import Corev1ScaleIOVolumeSource
        from ..models.corev_1_secret_volume_source import Corev1SecretVolumeSource
        from ..models.corev_1_storage_os_volume_source import (
            Corev1StorageOSVolumeSource,
        )
        from ..models.corev_1_vsphere_virtual_disk_volume_source import (
            Corev1VsphereVirtualDiskVolumeSource,
        )
        from ..models.corev_1aws_elastic_block_store_volume_source import (
            Corev1AWSElasticBlockStoreVolumeSource,
        )
        from ..models.corev_1csi_volume_source import Corev1CSIVolumeSource
        from ..models.corev_1fc_volume_source import Corev1FCVolumeSource
        from ..models.corev_1gce_persistent_disk_volume_source import (
            Corev1GCEPersistentDiskVolumeSource,
        )
        from ..models.corev_1iscsi_volume_source import Corev1ISCSIVolumeSource
        from ..models.corev_1nfs_volume_source import Corev1NFSVolumeSource
        from ..models.corev_1rbd_volume_source import Corev1RBDVolumeSource

        d = dict(src_dict)
        name = d.pop("name")

        def _parse_aws_elastic_block_store(
            data: object,
        ) -> Corev1AWSElasticBlockStoreVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                aws_elastic_block_store_type_1 = (
                    Corev1AWSElasticBlockStoreVolumeSource.from_dict(data)
                )

                return aws_elastic_block_store_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1AWSElasticBlockStoreVolumeSource | None | Unset, data)

        aws_elastic_block_store = _parse_aws_elastic_block_store(
            d.pop("awsElasticBlockStore", UNSET)
        )

        def _parse_azure_disk(
            data: object,
        ) -> Corev1AzureDiskVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                azure_disk_type_1 = Corev1AzureDiskVolumeSource.from_dict(data)

                return azure_disk_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1AzureDiskVolumeSource | None | Unset, data)

        azure_disk = _parse_azure_disk(d.pop("azureDisk", UNSET))

        def _parse_azure_file(
            data: object,
        ) -> Corev1AzureFileVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                azure_file_type_1 = Corev1AzureFileVolumeSource.from_dict(data)

                return azure_file_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1AzureFileVolumeSource | None | Unset, data)

        azure_file = _parse_azure_file(d.pop("azureFile", UNSET))

        def _parse_cephfs(data: object) -> Corev1CephFSVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                cephfs_type_1 = Corev1CephFSVolumeSource.from_dict(data)

                return cephfs_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1CephFSVolumeSource | None | Unset, data)

        cephfs = _parse_cephfs(d.pop("cephfs", UNSET))

        def _parse_cinder(data: object) -> Corev1CinderVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                cinder_type_1 = Corev1CinderVolumeSource.from_dict(data)

                return cinder_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1CinderVolumeSource | None | Unset, data)

        cinder = _parse_cinder(d.pop("cinder", UNSET))

        def _parse_config_map(
            data: object,
        ) -> Corev1ConfigMapVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                config_map_type_1 = Corev1ConfigMapVolumeSource.from_dict(data)

                return config_map_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1ConfigMapVolumeSource | None | Unset, data)

        config_map = _parse_config_map(d.pop("configMap", UNSET))

        def _parse_csi(data: object) -> Corev1CSIVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                csi_type_1 = Corev1CSIVolumeSource.from_dict(data)

                return csi_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1CSIVolumeSource | None | Unset, data)

        csi = _parse_csi(d.pop("csi", UNSET))

        def _parse_downward_api(
            data: object,
        ) -> Corev1DownwardAPIVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                downward_api_type_1 = Corev1DownwardAPIVolumeSource.from_dict(data)

                return downward_api_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1DownwardAPIVolumeSource | None | Unset, data)

        downward_api = _parse_downward_api(d.pop("downwardAPI", UNSET))

        def _parse_empty_dir(data: object) -> Corev1EmptyDirVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                empty_dir_type_1 = Corev1EmptyDirVolumeSource.from_dict(data)

                return empty_dir_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1EmptyDirVolumeSource | None | Unset, data)

        empty_dir = _parse_empty_dir(d.pop("emptyDir", UNSET))

        def _parse_ephemeral(
            data: object,
        ) -> Corev1EphemeralVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                ephemeral_type_1 = Corev1EphemeralVolumeSource.from_dict(data)

                return ephemeral_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1EphemeralVolumeSource | None | Unset, data)

        ephemeral = _parse_ephemeral(d.pop("ephemeral", UNSET))

        def _parse_fc(data: object) -> Corev1FCVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                fc_type_1 = Corev1FCVolumeSource.from_dict(data)

                return fc_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1FCVolumeSource | None | Unset, data)

        fc = _parse_fc(d.pop("fc", UNSET))

        def _parse_flex_volume(data: object) -> Corev1FlexVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                flex_volume_type_1 = Corev1FlexVolumeSource.from_dict(data)

                return flex_volume_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1FlexVolumeSource | None | Unset, data)

        flex_volume = _parse_flex_volume(d.pop("flexVolume", UNSET))

        def _parse_flocker(data: object) -> Corev1FlockerVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                flocker_type_1 = Corev1FlockerVolumeSource.from_dict(data)

                return flocker_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1FlockerVolumeSource | None | Unset, data)

        flocker = _parse_flocker(d.pop("flocker", UNSET))

        def _parse_gce_persistent_disk(
            data: object,
        ) -> Corev1GCEPersistentDiskVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                gce_persistent_disk_type_1 = (
                    Corev1GCEPersistentDiskVolumeSource.from_dict(data)
                )

                return gce_persistent_disk_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1GCEPersistentDiskVolumeSource | None | Unset, data)

        gce_persistent_disk = _parse_gce_persistent_disk(
            d.pop("gcePersistentDisk", UNSET)
        )

        def _parse_git_repo(data: object) -> Corev1GitRepoVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                git_repo_type_1 = Corev1GitRepoVolumeSource.from_dict(data)

                return git_repo_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1GitRepoVolumeSource | None | Unset, data)

        git_repo = _parse_git_repo(d.pop("gitRepo", UNSET))

        def _parse_glusterfs(
            data: object,
        ) -> Corev1GlusterfsVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                glusterfs_type_1 = Corev1GlusterfsVolumeSource.from_dict(data)

                return glusterfs_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1GlusterfsVolumeSource | None | Unset, data)

        glusterfs = _parse_glusterfs(d.pop("glusterfs", UNSET))

        def _parse_host_path(data: object) -> Corev1HostPathVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                host_path_type_1 = Corev1HostPathVolumeSource.from_dict(data)

                return host_path_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1HostPathVolumeSource | None | Unset, data)

        host_path = _parse_host_path(d.pop("hostPath", UNSET))

        def _parse_image(data: object) -> Corev1ImageVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                image_type_1 = Corev1ImageVolumeSource.from_dict(data)

                return image_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1ImageVolumeSource | None | Unset, data)

        image = _parse_image(d.pop("image", UNSET))

        def _parse_iscsi(data: object) -> Corev1ISCSIVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                iscsi_type_1 = Corev1ISCSIVolumeSource.from_dict(data)

                return iscsi_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1ISCSIVolumeSource | None | Unset, data)

        iscsi = _parse_iscsi(d.pop("iscsi", UNSET))

        def _parse_nfs(data: object) -> Corev1NFSVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                nfs_type_1 = Corev1NFSVolumeSource.from_dict(data)

                return nfs_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1NFSVolumeSource | None | Unset, data)

        nfs = _parse_nfs(d.pop("nfs", UNSET))

        def _parse_persistent_volume_claim(
            data: object,
        ) -> Corev1PersistentVolumeClaimVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                persistent_volume_claim_type_1 = (
                    Corev1PersistentVolumeClaimVolumeSource.from_dict(data)
                )

                return persistent_volume_claim_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1PersistentVolumeClaimVolumeSource | None | Unset, data)

        persistent_volume_claim = _parse_persistent_volume_claim(
            d.pop("persistentVolumeClaim", UNSET)
        )

        def _parse_photon_persistent_disk(
            data: object,
        ) -> Corev1PhotonPersistentDiskVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                photon_persistent_disk_type_1 = (
                    Corev1PhotonPersistentDiskVolumeSource.from_dict(data)
                )

                return photon_persistent_disk_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1PhotonPersistentDiskVolumeSource | None | Unset, data)

        photon_persistent_disk = _parse_photon_persistent_disk(
            d.pop("photonPersistentDisk", UNSET)
        )

        def _parse_portworx_volume(
            data: object,
        ) -> Corev1PortworxVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                portworx_volume_type_1 = Corev1PortworxVolumeSource.from_dict(data)

                return portworx_volume_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1PortworxVolumeSource | None | Unset, data)

        portworx_volume = _parse_portworx_volume(d.pop("portworxVolume", UNSET))

        def _parse_projected(
            data: object,
        ) -> Corev1ProjectedVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                projected_type_1 = Corev1ProjectedVolumeSource.from_dict(data)

                return projected_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1ProjectedVolumeSource | None | Unset, data)

        projected = _parse_projected(d.pop("projected", UNSET))

        def _parse_quobyte(data: object) -> Corev1QuobyteVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                quobyte_type_1 = Corev1QuobyteVolumeSource.from_dict(data)

                return quobyte_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1QuobyteVolumeSource | None | Unset, data)

        quobyte = _parse_quobyte(d.pop("quobyte", UNSET))

        def _parse_rbd(data: object) -> Corev1RBDVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                rbd_type_1 = Corev1RBDVolumeSource.from_dict(data)

                return rbd_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1RBDVolumeSource | None | Unset, data)

        rbd = _parse_rbd(d.pop("rbd", UNSET))

        def _parse_scale_io(data: object) -> Corev1ScaleIOVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                scale_io_type_1 = Corev1ScaleIOVolumeSource.from_dict(data)

                return scale_io_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1ScaleIOVolumeSource | None | Unset, data)

        scale_io = _parse_scale_io(d.pop("scaleIO", UNSET))

        def _parse_secret(data: object) -> Corev1SecretVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                secret_type_1 = Corev1SecretVolumeSource.from_dict(data)

                return secret_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1SecretVolumeSource | None | Unset, data)

        secret = _parse_secret(d.pop("secret", UNSET))

        def _parse_storageos(
            data: object,
        ) -> Corev1StorageOSVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                storageos_type_1 = Corev1StorageOSVolumeSource.from_dict(data)

                return storageos_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1StorageOSVolumeSource | None | Unset, data)

        storageos = _parse_storageos(d.pop("storageos", UNSET))

        def _parse_vsphere_volume(
            data: object,
        ) -> Corev1VsphereVirtualDiskVolumeSource | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                vsphere_volume_type_1 = Corev1VsphereVirtualDiskVolumeSource.from_dict(
                    data
                )

                return vsphere_volume_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(Corev1VsphereVirtualDiskVolumeSource | None | Unset, data)

        vsphere_volume = _parse_vsphere_volume(d.pop("vsphereVolume", UNSET))

        corev_1_volume = cls(
            name=name,
            aws_elastic_block_store=aws_elastic_block_store,
            azure_disk=azure_disk,
            azure_file=azure_file,
            cephfs=cephfs,
            cinder=cinder,
            config_map=config_map,
            csi=csi,
            downward_api=downward_api,
            empty_dir=empty_dir,
            ephemeral=ephemeral,
            fc=fc,
            flex_volume=flex_volume,
            flocker=flocker,
            gce_persistent_disk=gce_persistent_disk,
            git_repo=git_repo,
            glusterfs=glusterfs,
            host_path=host_path,
            image=image,
            iscsi=iscsi,
            nfs=nfs,
            persistent_volume_claim=persistent_volume_claim,
            photon_persistent_disk=photon_persistent_disk,
            portworx_volume=portworx_volume,
            projected=projected,
            quobyte=quobyte,
            rbd=rbd,
            scale_io=scale_io,
            secret=secret,
            storageos=storageos,
            vsphere_volume=vsphere_volume,
        )

        corev_1_volume.additional_properties = d
        return corev_1_volume

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
