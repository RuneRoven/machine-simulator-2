package config

// DefaultGlobalConfig returns the built-in default global configuration.
func DefaultGlobalConfig() GlobalConfig {
	return GlobalConfig{
		TimeScale:    1.0,
		RandomFactor: 0.1,
		TickRateMS:   100,
		LogLevel:     "info",
		API: APIConfig{
			Host: "0.0.0.0",
			Port: 8081,
		},
		ERP: ERPConfig{
			Mode:                "auto",
			AutoOrderIntervalMS: 30000,
			MaxBacklog:          5,
		},
		MES: MESConfig{
			DispatchIntervalMS: 1000,
			DispatchStrategy:   "round-robin",
		},
	}
}

// DefaultOPCUAAdapterConfig returns the built-in default OPC-UA adapter configuration.
func DefaultOPCUAAdapterConfig() OPCUAAdapterConfig {
	return OPCUAAdapterConfig{
		Enabled:      true,
		BasePort:     4840,
		UpdateRateMS: 100,
		MaxSessions:  100,
		Hostname:     "localhost",
	}
}

// DefaultModbusAdapterConfig returns the built-in default Modbus adapter configuration.
func DefaultModbusAdapterConfig() ModbusAdapterConfig {
	return ModbusAdapterConfig{
		Enabled:          true,
		Port:             502,
		MaxClients:       10,
		UpdateIntervalMS: 1000,
	}
}

// DefaultMachineDefaults returns fallback machine parameters when not specified.
func DefaultMachineDefaults() MachineDefaultsConfig {
	return MachineDefaultsConfig{
		CycleTimeMS:          5000,
		ScrapRate:            0.02,
		BreakdownProbability: 0.001,
		AutoRestoreTimeMS:    60000,
	}
}
