package hlims

// ProductKind identifies the specifications attached to a product.
type ProductKind string

const (
	ProductKindSystem         ProductKind = "system"
	ProductKindProcessor      ProductKind = "processor"
	ProductKindMemory         ProductKind = "memory"
	ProductKindDrive          ProductKind = "drive"
	ProductKindRack           ProductKind = "rack"
	ProductKindRouter         ProductKind = "router"
	ProductKindSwitch         ProductKind = "switch"
	ProductKindAccessPoint    ProductKind = "access_point"
	ProductKindNetworkAdapter ProductKind = "network_adapter"
)

// Valid reports whether the product kind is supported.
func (kind ProductKind) Valid() bool {
	switch kind {
	case ProductKindSystem, ProductKindProcessor, ProductKindMemory, ProductKindDrive,
		ProductKindRack, ProductKindRouter, ProductKindSwitch, ProductKindAccessPoint,
		ProductKindNetworkAdapter:
		return true
	default:
		return false
	}
}

// MachineKind identifies whether a machine runs on hardware or virtualization.
type MachineKind string

const (
	MachineKindBareMetal      MachineKind = "bare_metal"
	MachineKindVirtualMachine MachineKind = "virtual_machine"
)

// Valid reports whether the machine kind is supported.
func (kind MachineKind) Valid() bool {
	return kind == MachineKindBareMetal || kind == MachineKindVirtualMachine
}

// NetworkKind identifies the current network scopes represented by HLIMS.
type NetworkKind string

const (
	NetworkKindLAN      NetworkKind = "lan"
	NetworkKindTailnet  NetworkKind = "tailnet"
	NetworkKindCloudVPC NetworkKind = "cloud_vpc"
	NetworkKindPublic   NetworkKind = "public"
	NetworkKindLoopback NetworkKind = "loopback"
)

// Valid reports whether the network kind is supported.
func (kind NetworkKind) Valid() bool {
	switch kind {
	case NetworkKindLAN, NetworkKindTailnet, NetworkKindCloudVPC, NetworkKindPublic, NetworkKindLoopback:
		return true
	default:
		return false
	}
}

// CPUAllocationKind identifies whether processor time is shared or dedicated.
type CPUAllocationKind string

const (
	CPUAllocationShared    CPUAllocationKind = "shared"
	CPUAllocationDedicated CPUAllocationKind = "dedicated"
)

// Valid reports whether the CPU allocation kind is supported.
func (kind CPUAllocationKind) Valid() bool {
	return kind == CPUAllocationShared || kind == CPUAllocationDedicated
}

// StorageMediaKind identifies the storage medium.
type StorageMediaKind string

const (
	StorageMediaHDD StorageMediaKind = "hdd"
	StorageMediaSSD StorageMediaKind = "ssd"
)

// Valid reports whether the storage media kind is supported.
func (kind StorageMediaKind) Valid() bool {
	return kind == StorageMediaHDD || kind == StorageMediaSSD
}

// StorageInterfaceKind identifies how storage connects or is presented.
type StorageInterfaceKind string

const (
	StorageInterfaceSATA    StorageInterfaceKind = "sata"
	StorageInterfaceSAS     StorageInterfaceKind = "sas"
	StorageInterfaceNVMe    StorageInterfaceKind = "nvme"
	StorageInterfaceUSB     StorageInterfaceKind = "usb"
	StorageInterfaceSCSI    StorageInterfaceKind = "scsi"
	StorageInterfaceVirtIO  StorageInterfaceKind = "virtio"
	StorageInterfaceVirtual StorageInterfaceKind = "virtual"
)

// Valid reports whether the storage interface kind is supported.
func (kind StorageInterfaceKind) Valid() bool {
	switch kind {
	case StorageInterfaceSATA, StorageInterfaceSAS, StorageInterfaceNVMe,
		StorageInterfaceUSB, StorageInterfaceSCSI, StorageInterfaceVirtIO,
		StorageInterfaceVirtual:
		return true
	default:
		return false
	}
}

// InstanceScheme identifies the URL scheme used to reach a service instance.
type InstanceScheme string

const (
	InstanceSchemeHTTP  InstanceScheme = "http"
	InstanceSchemeHTTPS InstanceScheme = "https"
)

// Valid reports whether the instance scheme is supported.
func (scheme InstanceScheme) Valid() bool {
	return scheme == InstanceSchemeHTTP || scheme == InstanceSchemeHTTPS
}
