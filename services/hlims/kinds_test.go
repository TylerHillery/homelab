package hlims

import "testing"

func TestKinds(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{name: "product system", valid: ProductKindSystem.Valid()},
		{name: "product processor", valid: ProductKindProcessor.Valid()},
		{name: "product memory", valid: ProductKindMemory.Valid()},
		{name: "product drive", valid: ProductKindDrive.Valid()},
		{name: "machine bare metal", valid: MachineKindBareMetal.Valid()},
		{name: "machine virtual", valid: MachineKindVirtualMachine.Valid()},
		{name: "network LAN", valid: NetworkKindLAN.Valid()},
		{name: "network tailnet", valid: NetworkKindTailnet.Valid()},
		{name: "network cloud VPC", valid: NetworkKindCloudVPC.Valid()},
		{name: "network public", valid: NetworkKindPublic.Valid()},
		{name: "CPU shared", valid: CPUAllocationShared.Valid()},
		{name: "CPU dedicated", valid: CPUAllocationDedicated.Valid()},
		{name: "storage HDD", valid: StorageMediaHDD.Valid()},
		{name: "storage SSD", valid: StorageMediaSSD.Valid()},
		{name: "storage SATA", valid: StorageInterfaceSATA.Valid()},
		{name: "storage SAS", valid: StorageInterfaceSAS.Valid()},
		{name: "storage NVMe", valid: StorageInterfaceNVMe.Valid()},
		{name: "storage USB", valid: StorageInterfaceUSB.Valid()},
		{name: "storage SCSI", valid: StorageInterfaceSCSI.Valid()},
		{name: "storage VirtIO", valid: StorageInterfaceVirtIO.Valid()},
		{name: "storage virtual", valid: StorageInterfaceVirtual.Valid()},
		{name: "instance HTTP", valid: InstanceSchemeHTTP.Valid()},
		{name: "instance HTTPS", valid: InstanceSchemeHTTPS.Valid()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.valid {
				t.Fatal("Valid() = false; want true")
			}
		})
	}

	if ProductKind("invalid").Valid() {
		t.Error("invalid product kind is valid")
	}
	if MachineKind("invalid").Valid() {
		t.Error("invalid machine kind is valid")
	}
	if NetworkKind("invalid").Valid() {
		t.Error("invalid network kind is valid")
	}
	if CPUAllocationKind("invalid").Valid() {
		t.Error("invalid CPU allocation kind is valid")
	}
	if StorageMediaKind("invalid").Valid() {
		t.Error("invalid storage media kind is valid")
	}
	if StorageInterfaceKind("invalid").Valid() {
		t.Error("invalid storage interface kind is valid")
	}
	if InstanceScheme("invalid").Valid() {
		t.Error("invalid instance scheme is valid")
	}
}
