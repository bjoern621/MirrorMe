package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"bjoernblessin.de/screenshare/internal/gpu"
	"bjoernblessin.de/screenshare/internal/qa"
)

func platform() qa.Machine {
	return qa.Machine{
		OS:       osRelease(),
		Kernel:   strings.TrimSpace(readFile("/proc/sys/kernel/osrelease")),
		CPU:      cpuModel(),
		GPUs:     gpus(),
		VADriver: gpu.Detect().Vendor,
	}
}

func osRelease() string {
	for line := range strings.SplitSeq(readFile("/etc/os-release"), "\n") {
		if value, found := strings.CutPrefix(line, "PRETTY_NAME="); found {
			return strings.Trim(value, `"`)
		}
	}
	return ""
}

// cpuModel reads the first of "model name" (x86), "Model" and "Hardware"
// (ARM boards such as a Raspberry Pi) in /proc/cpuinfo.
func cpuModel() string {
	info := readFile("/proc/cpuinfo")
	for _, key := range []string{"model name", "Model", "Hardware"} {
		for line := range strings.SplitSeq(info, "\n") {
			name, value, found := strings.Cut(line, ":")
			if found && strings.TrimSpace(name) == key {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

// card matches a DRM card node and leaves out its connectors ("card1-DP-1").
var card = regexp.MustCompile(`^card\d+$`)

// gpus lists every DRM card, named by lspci for a PCI device
// and by its devicetree compatible string for one on a board, such as a Raspberry Pi's.
func gpus() []qa.GPU {
	entries, _ := os.ReadDir("/sys/class/drm")
	var out []qa.GPU
	for _, e := range entries {
		if !card.MatchString(e.Name()) {
			continue
		}
		device := filepath.Join("/sys/class/drm", e.Name(), "device")
		link, _ := os.Readlink(filepath.Join(device, "driver"))
		driver := filepath.Base(link)
		uevent := readFile(filepath.Join(device, "uevent"))

		g := qa.GPU{Driver: driver, DriverVersion: driverVersion(driver)}
		if slot := ueventValue(uevent, "PCI_SLOT_NAME"); slot != "" {
			g.Name = pciName(slot)
		} else {
			g.Name = ueventValue(uevent, "OF_COMPATIBLE_0")
		}
		if g.Name == "" {
			g.Name = driver
		}
		out = append(out, g)
	}
	return out
}

// pciName answers lspci's description of the device at slot, without its class:
// "Advanced Micro Devices, Inc. [AMD/ATI] Phoenix1 (rev c6)".
func pciName(slot string) string {
	if _, err := exec.LookPath("lspci"); err != nil {
		return "PCI " + slot
	}
	_, name, _ := strings.Cut(strings.TrimSpace(output("lspci", "-s", slot)), ": ")
	return name
}

// driverVersion answers what a kernel driver reports of its release.
// NVIDIA's names it in /proc. An in-tree driver ships with the kernel and reports none.
func driverVersion(driver string) string {
	if driver == "nvidia" {
		fields := strings.Fields(readFile("/proc/driver/nvidia/version"))
		for i, f := range fields {
			if f == "Module" && i+1 < len(fields) {
				return fields[i+1]
			}
		}
	}
	return strings.TrimSpace(readFile(filepath.Join("/sys/module", driver, "version")))
}

func ueventValue(uevent, key string) string {
	for line := range strings.SplitSeq(uevent, "\n") {
		if value, found := strings.CutPrefix(line, key+"="); found {
			return value
		}
	}
	return ""
}

// readFile answers the file's contents, or "" where the machine has no such file.
func readFile(path string) string {
	data, _ := os.ReadFile(path)
	return string(data)
}
