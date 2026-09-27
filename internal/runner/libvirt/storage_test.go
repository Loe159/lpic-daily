package libvirt

import (
	"testing"
)

func TestParseStorageSnapshotFlattensDevicesAndMarksActiveSwap(t *testing.T) {
	lsblk := []byte(`{
  "blockdevices": [
    {
      "path": "/dev/vdb",
      "type": "disk",
      "fstype": null,
      "mountpoint": null,
      "pttype": "gpt",
      "size": 536870912,
      "children": [
        {
          "path": "/dev/vdb1",
          "type": "part",
          "fstype": "ext4",
          "mountpoint": "/srv/data",
          "pttype": null,
          "size": "402653184"
        },
        {
          "path": "/dev/vdb2",
          "type": "part",
          "fstype": "swap",
          "mountpoint": null,
          "pttype": null,
          "size": 134217728
        }
      ]
    }
  ]
}`)
	swaps := []byte("Filename\tType\tSize\tUsed\tPriority\n/dev/vdb2 partition 131068 0 -2\n")

	snapshot, err := parseStorageSnapshot(lsblk, swaps)
	if err != nil {
		t.Fatalf("parseStorageSnapshot() error = %v", err)
	}
	if len(snapshot.Devices) != 3 {
		t.Fatalf("devices = %d, want 3", len(snapshot.Devices))
	}

	disk := snapshot.Devices[0]
	if disk.Path != "/dev/vdb" ||
		disk.DeviceType != "disk" ||
		disk.PartitionTable != "gpt" ||
		disk.SizeBytes != 536870912 {
		t.Fatalf("disk = %#v", disk)
	}

	data := snapshot.Devices[1]
	if data.Path != "/dev/vdb1" ||
		data.Filesystem != "ext4" ||
		data.Mountpoint != "/srv/data" ||
		data.SwapActive {
		t.Fatalf("data partition = %#v", data)
	}

	swap := snapshot.Devices[2]
	if swap.Path != "/dev/vdb2" ||
		swap.Filesystem != "swap" ||
		!swap.SwapActive ||
		swap.SizeBytes != 134217728 {
		t.Fatalf("swap partition = %#v", swap)
	}
}

func TestParseStorageSnapshotRejectsMissingDevicePath(t *testing.T) {
	_, err := parseStorageSnapshot(
		[]byte(`{"blockdevices":[{"path":"","type":"disk","size":"1"}]}`),
		[]byte("Filename Type Size Used Priority\n"),
	)
	if err == nil {
		t.Fatal("missing device path unexpectedly accepted")
	}
}
