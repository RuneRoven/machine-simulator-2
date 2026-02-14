package protocol

import (
	"context"

	"machine-simulator-2/internal/machine"
)

// Adapter is the interface that protocol adapters must implement.
type Adapter interface {
	// Name returns the adapter name (e.g., "opcua", "modbus").
	Name() string

	// Start begins the adapter (servers, listeners, etc).
	Start(ctx context.Context) error

	// Stop gracefully shuts down the adapter.
	Stop() error

	// RegisterMachine adds a machine to be exposed via this protocol.
	RegisterMachine(m machine.Machine, cfg AdapterMachineConfig) error

	// DeregisterMachine removes a machine from this protocol.
	DeregisterMachine(machineID string) error
}

// AdapterMachineConfig holds per-machine adapter settings.
type AdapterMachineConfig struct {
	Port     int               `yaml:"port"`
	UnitID   int               `yaml:"unit_id"`
	Metadata map[string]string `yaml:"metadata"`
}
