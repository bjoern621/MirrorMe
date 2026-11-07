package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"bjoernblessin.de/screenshare/internal/qa"
)

// inventory asks CIM for the OS, the first CPU and every display adapter, as one JSON object.
const inventory = `$o = Get-CimInstance Win32_OperatingSystem
$c = Get-CimInstance Win32_Processor | Select-Object -First 1
$g = @(Get-CimInstance Win32_VideoController | ForEach-Object { @{ name = $_.Name; version = $_.DriverVersion } })
@{ os = "$($o.Caption) $($o.Version)"; cpu = $c.Name.Trim(); gpus = $g } | ConvertTo-Json -Depth 3 -Compress`

func platform() qa.Machine {
	out := output("powershell", "-NoProfile", "-NonInteractive", "-Command", inventory)
	if out == "" {
		return qa.Machine{}
	}

	var answer struct {
		OS   string          `json:"os"`
		CPU  string          `json:"cpu"`
		GPUs json.RawMessage `json:"gpus"`
	}
	if err := json.Unmarshal([]byte(out), &answer); err != nil {
		fmt.Fprintf(os.Stderr, "warning: reading the CIM inventory: %v, leaving it out of the record\n", err)
		return qa.Machine{}
	}
	return qa.Machine{OS: strings.TrimSpace(answer.OS), CPU: answer.CPU, GPUs: adapters(answer.GPUs)}
}

type adapter struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// adapters reads the adapter list, which ConvertTo-Json writes as a bare object for a single adapter.
func adapters(raw json.RawMessage) []qa.GPU {
	var list []adapter
	if err := json.Unmarshal(raw, &list); err != nil {
		var one adapter
		if json.Unmarshal(raw, &one) != nil {
			return nil
		}
		list = []adapter{one}
	}

	var out []qa.GPU
	for _, a := range list {
		out = append(out, qa.GPU{Name: a.Name, DriverVersion: a.Version})
	}
	return out
}
