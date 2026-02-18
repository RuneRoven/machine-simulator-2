package config

// Config is the fully assembled configuration after loading profile, templates, and overrides.
type Config struct {
	Global    GlobalConfig
	Profile   ProfileConfig
	Templates map[string]*LineTemplateConfig // keyed by template name
	Adapters  AdaptersConfig
	Lines     []LineInstanceConfig // instantiated from profile + templates
}

// GlobalConfig holds simulator-wide settings.
type GlobalConfig struct {
	TimeScale    float64 `yaml:"time_scale"    env:"SIMULATOR_TIME_SCALE"`
	RandomFactor float64 `yaml:"random_factor" env:"SIMULATOR_RANDOM_FACTOR"`
	TickRateMS   int     `yaml:"tick_rate_ms"  env:"SIMULATOR_TICK_RATE_MS"`
	LogLevel     string  `yaml:"log_level"     env:"SIMULATOR_LOG_LEVEL"`

	API     APIConfig     `yaml:"api"`
	ERP     ERPConfig     `yaml:"erp"`
	MES     MESConfig     `yaml:"mes"`
	Webhook WebhookConfig `yaml:"webhook"`
}

// WebhookConfig holds webhook publisher settings.
type WebhookConfig struct {
	Enabled       bool     `yaml:"enabled"        env:"SIMULATOR_WEBHOOK_ENABLED"`
	TargetURL     string   `yaml:"target_url"     env:"SIMULATOR_WEBHOOK_TARGET_URL"`
	EventTypes    []string `yaml:"event_types"    env:"SIMULATOR_WEBHOOK_EVENT_TYPES"`
	TimeoutMS     int      `yaml:"timeout_ms"     env:"SIMULATOR_WEBHOOK_TIMEOUT_MS"`
	RetryAttempts int      `yaml:"retry_attempts"`
	RetryDelayMS  int      `yaml:"retry_delay_ms"`
}

// APIConfig holds HTTP server settings.
type APIConfig struct {
	Host string `yaml:"host" env:"SIMULATOR_API_HOST"`
	Port int    `yaml:"port" env:"SIMULATOR_API_PORT"`
}

// ERPConfig holds ERP simulator settings.
type ERPConfig struct {
	Mode                 string `yaml:"mode"                   env:"SIMULATOR_ERP_MODE"`
	AutoOrderIntervalMS  int    `yaml:"auto_order_interval_ms" env:"SIMULATOR_ERP_AUTO_ORDER_INTERVAL_MS"`
	MaxBacklog           int    `yaml:"max_backlog"            env:"SIMULATOR_ERP_MAX_BACKLOG"`
}

// MESConfig holds MES simulator settings.
type MESConfig struct {
	DispatchIntervalMS int    `yaml:"dispatch_interval_ms" env:"SIMULATOR_MES_DISPATCH_INTERVAL_MS"`
	DispatchStrategy   string `yaml:"dispatch_strategy"    env:"SIMULATOR_MES_DISPATCH_STRATEGY"`
}

// ProfileConfig represents a named factory profile.
type ProfileConfig struct {
	Name        string              `yaml:"name"`
	Description string              `yaml:"description"`
	Lines       []ProfileLineEntry  `yaml:"lines"`
	Overrides   ProfileOverrides    `yaml:"overrides"`
}

// ProfileLineEntry specifies a line template and how many instances.
type ProfileLineEntry struct {
	Template  string `yaml:"template"`
	Instances int    `yaml:"instances"`
}

// ProfileOverrides allows a profile to override global settings.
type ProfileOverrides struct {
	Global map[string]interface{} `yaml:"global"`
}

// LineTemplateConfig represents a line template loaded from config/lines/.
type LineTemplateConfig struct {
	Name        string               `yaml:"name"`
	Domain      string               `yaml:"domain"`
	Description string               `yaml:"description"`
	Machines    []MachineStageConfig  `yaml:"machines"`
	Buffers     BufferTemplateConfig  `yaml:"buffers"`
	Recipes     []RecipeConfig        `yaml:"recipes"`
}

// MachineStageConfig defines a machine stage within a line template.
type MachineStageConfig struct {
	Stage    int                    `yaml:"stage"`
	Type     string                 `yaml:"type"`
	Role     string                 `yaml:"role"`
	Defaults MachineDefaultsConfig  `yaml:"defaults"`
}

// MachineDefaultsConfig holds default operating parameters for a machine stage.
type MachineDefaultsConfig struct {
	CycleTimeMS          int     `yaml:"cycle_time_ms"`
	ScrapRate            float64 `yaml:"scrap_rate"`
	BreakdownProbability float64 `yaml:"breakdown_probability"`
	AutoRestoreTimeMS    int     `yaml:"auto_restore_time_ms"`
}

// BufferTemplateConfig holds buffer settings for a line template.
type BufferTemplateConfig struct {
	DefaultCapacity int                    `yaml:"default_capacity"`
	Overrides       []BufferOverrideConfig `yaml:"overrides"`
}

// BufferOverrideConfig overrides the buffer capacity between two specific stages.
type BufferOverrideConfig struct {
	FromStage int `yaml:"from_stage"`
	ToStage   int `yaml:"to_stage"`
	Capacity  int `yaml:"capacity"`
}

// RecipeConfig defines a product recipe within a line template.
type RecipeConfig struct {
	ProductID      string                           `yaml:"product_id"`
	Description    string                           `yaml:"description"`
	PlannedQtyRange [2]int                          `yaml:"planned_qty_range"`
	Overrides      map[int]RecipeStageOverrideConfig `yaml:"overrides"`
}

// RecipeStageOverrideConfig holds per-stage overrides for a recipe.
type RecipeStageOverrideConfig struct {
	CycleTimeMS          *int     `yaml:"cycle_time_ms,omitempty"`
	ScrapRate            *float64 `yaml:"scrap_rate,omitempty"`
	BreakdownProbability *float64 `yaml:"breakdown_probability,omitempty"`
}

// AdaptersConfig holds all protocol adapter configurations.
type AdaptersConfig struct {
	OPCUA  OPCUAAdapterConfig  `yaml:"opcua"`
	Modbus ModbusAdapterConfig `yaml:"modbus"`
}

// OPCUAAdapterConfig holds OPC-UA adapter settings.
type OPCUAAdapterConfig struct {
	Enabled      bool   `yaml:"enabled"        env:"SIMULATOR_OPCUA_ENABLED"`
	BasePort     int    `yaml:"base_port"      env:"SIMULATOR_OPCUA_BASE_PORT"`
	UpdateRateMS int    `yaml:"update_rate_ms" env:"SIMULATOR_OPCUA_UPDATE_RATE_MS"`
	MaxSessions  int    `yaml:"max_sessions"   env:"SIMULATOR_OPCUA_MAX_SESSIONS"`
	Hostname     string `yaml:"hostname"       env:"SIMULATOR_OPCUA_HOSTNAME"`
}

// ModbusAdapterConfig holds Modbus TCP adapter settings.
type ModbusAdapterConfig struct {
	Enabled          bool `yaml:"enabled"            env:"SIMULATOR_MODBUS_ENABLED"`
	Port             int  `yaml:"port"               env:"SIMULATOR_MODBUS_PORT"`
	MaxClients       int  `yaml:"max_clients"        env:"SIMULATOR_MODBUS_MAX_CLIENTS"`
	UpdateIntervalMS int  `yaml:"update_interval_ms" env:"SIMULATOR_MODBUS_UPDATE_INTERVAL_MS"`
}

// LineInstanceConfig is the assembled config for a running line instance.
type LineInstanceConfig struct {
	InstanceID   string
	TemplateName string
	Domain       string
	Machines     []MachineInstanceConfig
	Buffers      []BufferInstanceConfig
	Recipes      []RecipeConfig
}

// MachineInstanceConfig is the assembled config for a single machine within a line instance.
type MachineInstanceConfig struct {
	ID                   string
	Type                 string
	Role                 string
	LineInstanceID       string
	Stage                int
	CycleTimeMS          int
	ScrapRate            float64
	BreakdownProbability float64
	AutoRestoreTimeMS    int
}

// BufferInstanceConfig is the assembled config for a buffer within a line instance.
type BufferInstanceConfig struct {
	ID                  string
	UpstreamMachineID   string
	DownstreamMachineID string
	Capacity            int
}
