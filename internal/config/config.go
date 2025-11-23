package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/EmekaIwuagwu/articium-hub/internal/types"
	"github.com/spf13/viper"
)

// Config represents the main application configuration
type Config struct {
	Environment types.Environment   `mapstructure:"environment"`
	Server      ServerConfig        `mapstructure:"server"`
	Chains      []types.ChainConfig `mapstructure:"chains"`
	Database    DatabaseConfig      `mapstructure:"database"`
	Queue       QueueConfig         `mapstructure:"queue"`
	Cache       CacheConfig         `mapstructure:"cache"`
	Relayer     RelayerConfig       `mapstructure:"relayer"`
	Security    SecurityConfig      `mapstructure:"security"`
	Crypto      CryptoConfig        `mapstructure:"crypto"`
	Monitoring  MonitoringConfig    `mapstructure:"monitoring"`
	Alerting    AlertingConfig      `mapstructure:"alerting"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host           string `mapstructure:"host"`
	Port           int    `mapstructure:"port"`
	TLSEnabled     bool   `mapstructure:"tls_enabled"`
	TLSCertPath    string `mapstructure:"tls_cert_path"`
	TLSKeyPath     string `mapstructure:"tls_key_path"`
	ReadTimeout    string `mapstructure:"read_timeout"`
	WriteTimeout   string `mapstructure:"write_timeout"`
	MaxHeaderBytes int    `mapstructure:"max_header_bytes"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Database     string `mapstructure:"database"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	SSLMode      string `mapstructure:"ssl_mode"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxLifetime  string `mapstructure:"max_lifetime"`
}

// QueueConfig represents message queue configuration
type QueueConfig struct {
	Type       string   `mapstructure:"type"` // nats, kafka, redis
	URLs       []string `mapstructure:"urls"`
	Subject    string   `mapstructure:"subject"`
	StreamName string   `mapstructure:"stream_name"`
	MaxRetries int      `mapstructure:"max_retries"`
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	Type      string   `mapstructure:"type"` // redis, memcached
	Addresses []string `mapstructure:"addresses"`
	Password  string   `mapstructure:"password"`
	DB        int      `mapstructure:"db"`
	TTL       string   `mapstructure:"ttl"`
}

// RelayerConfig represents relayer configuration
type RelayerConfig struct {
	Workers                 int    `mapstructure:"workers"`
	MaxRetries              int    `mapstructure:"max_retries"`
	RetryBackoff            string `mapstructure:"retry_backoff"`
	ProcessingTimeout       string `mapstructure:"processing_timeout"`
	EnableCircuitBreaker    bool   `mapstructure:"enable_circuit_breaker"`
	CircuitBreakerThreshold int    `mapstructure:"circuit_breaker_threshold"`
	BatchSize               int    `mapstructure:"batch_size"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	RequiredSignatures        int      `mapstructure:"required_signatures"`
	ValidatorAddresses        []string `mapstructure:"validator_addresses"`
	MaxTransactionAmount      string   `mapstructure:"max_transaction_amount"`
	DailyVolumeLimit          string   `mapstructure:"daily_volume_limit"`
	EnableRateLimiting        bool     `mapstructure:"enable_rate_limiting"`
	RateLimitPerHour          int      `mapstructure:"rate_limit_per_hour"`
	RateLimitPerAddress       int      `mapstructure:"rate_limit_per_address"`
	EnableEmergencyPause      bool     `mapstructure:"enable_emergency_pause"`
	EnableFraudDetection      bool     `mapstructure:"enable_fraud_detection"`
	AlertingWebhook           string   `mapstructure:"alerting_webhook"`
	LargeTransactionThreshold string   `mapstructure:"large_transaction_threshold"`
}

// CryptoConfig represents cryptography configuration
type CryptoConfig struct {
	EVMKeystorePath    string            `mapstructure:"evm_keystore_path"`
	SolanaKeystorePath string            `mapstructure:"solana_keystore_path"`
	NEARKeystorePath   string            `mapstructure:"near_keystore_path"`
	PasswordEnvVar     string            `mapstructure:"password_env_var"`
	UseAWSKMS          bool              `mapstructure:"use_aws_kms"`
	KMSKeyIDs          map[string]string `mapstructure:"kms_key_ids"`
}

// MonitoringConfig represents monitoring configuration
type MonitoringConfig struct {
	PrometheusPort        int    `mapstructure:"prometheus_port"`
	EnableTracing         bool   `mapstructure:"enable_tracing"`
	JaegerEndpoint        string `mapstructure:"jaeger_endpoint"`
	LogLevel              string `mapstructure:"log_level"`
	EnableMetricsExport   bool   `mapstructure:"enable_metrics_export"`
	MetricsExportInterval string `mapstructure:"metrics_export_interval"`
}

// AlertingConfig represents alerting configuration
type AlertingConfig struct {
	Enabled                   bool   `mapstructure:"enabled"`
	SlackWebhook              string `mapstructure:"slack_webhook"`
	PagerDutyKey              string `mapstructure:"pagerduty_key"`
	AlertOnFailureThreshold   int    `mapstructure:"alert_on_failure_threshold"`
	AlertOnHighGas            bool   `mapstructure:"alert_on_high_gas"`
	AlertOnLargeTransaction   bool   `mapstructure:"alert_on_large_transaction"`
	LargeTransactionThreshold string `mapstructure:"large_transaction_threshold"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	// Set default config path if not provided
	if configPath == "" {
		env := os.Getenv("BRIDGE_ENVIRONMENT")
		if env == "" {
			env = "development"
		}
		configPath = getConfigPathForEnv(env)
	}

	viper.SetConfigFile(configPath)

	// Allow environment variable overrides
	viper.AutomaticEnv()
	viper.SetEnvPrefix("BRIDGE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := ValidateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// getConfigPathForEnv returns the config file path for the given environment
func getConfigPathForEnv(env string) string {
	switch env {
	case "mainnet":
		return "config/config.mainnet.yaml"
	case "testnet":
		return "config/config.testnet.yaml"
	default:
		return "config/config.dev.yaml"
	}
}

// ValidateConfig validates the configuration
func ValidateConfig(config *Config) error {
	// Validate environment
	if config.Environment == "" {
		return fmt.Errorf("environment must be specified")
	}

	// Validate chains
	if len(config.Chains) == 0 {
		return fmt.Errorf("at least one chain must be configured")
	}

	for i, chain := range config.Chains {
		if err := validateChainConfig(&chain, config.Environment); err != nil {
			return fmt.Errorf("invalid chain config at index %d (%s): %w", i, chain.Name, err)
		}
	}

	// Validate mainnet security requirements
	if config.Environment == types.EnvironmentMainnet {
		if err := validateMainnetSecurity(&config.Security); err != nil {
			return fmt.Errorf("mainnet security validation failed: %w", err)
		}
	}

	// Validate database config
	if config.Database.Host == "" {
		return fmt.Errorf("database host must be specified")
	}

	// Validate relayer config
	if config.Relayer.Workers < 1 {
		return fmt.Errorf("relayer workers must be at least 1")
	}
	if config.Relayer.Workers > 50 {
		return fmt.Errorf("relayer workers should not exceed 50")
	}

	return nil
}

// validateChainConfig validates a single chain configuration
func validateChainConfig(chain *types.ChainConfig, env types.Environment) error {
	if chain.Name == "" {
		return fmt.Errorf("chain name is required")
	}

	if chain.ChainType == "" {
		return fmt.Errorf("chain type is required")
	}

	// Verify environment matches
	if chain.Environment != env {
		return fmt.Errorf("chain environment (%s) does not match global environment (%s)",
			chain.Environment, env)
	}

	if len(chain.RPCEndpoints) == 0 {
		return fmt.Errorf("at least one RPC endpoint is required")
	}

	// Validate chain-specific fields
	switch chain.ChainType {
	case types.ChainTypeEVM:
		if chain.ChainID == "" {
			return fmt.Errorf("EVM chain must have chain_id")
		}
		if chain.BridgeContract == "" {
			return fmt.Errorf("EVM chain must have bridge_contract")
		}

	case types.ChainTypeSolana:
		if chain.BridgeProgram == "" {
			return fmt.Errorf("Solana chain must have bridge_program")
		}
		if chain.Commitment == "" {
			chain.Commitment = "finalized" // default
		}

	case types.ChainTypeNEAR:
		if chain.NetworkID == "" {
			return fmt.Errorf("NEAR chain must have network_id")
		}
		if chain.BridgeContract == "" {
			return fmt.Errorf("NEAR chain must have bridge_contract")
		}

	case types.ChainTypeAlgorand:
		if chain.NetworkID == "" {
			return fmt.Errorf("Algorand chain must have network_id")
		}
		if chain.BridgeContract == "" {
			return fmt.Errorf("Algorand chain must have bridge_contract (application ID)")
		}

	case types.ChainTypeAptos:
		if chain.NetworkID == "" {
			return fmt.Errorf("Aptos chain must have network_id")
		}
		if chain.BridgeContract == "" {
			return fmt.Errorf("Aptos chain must have bridge_contract (module address)")
		}

	default:
		return fmt.Errorf("unsupported chain type: %s", chain.ChainType)
	}

	return nil
}

// validateMainnetSecurity validates mainnet-specific security requirements
func validateMainnetSecurity(security *SecurityConfig) error {
	if security.RequiredSignatures < 3 {
		return fmt.Errorf("mainnet requires at least 3 signatures (3-of-5 minimum)")
	}

	if len(security.ValidatorAddresses) < 5 {
		return fmt.Errorf("mainnet requires at least 5 validator addresses")
	}

	if !security.EnableEmergencyPause {
		return fmt.Errorf("mainnet requires emergency pause to be enabled")
	}

	if !security.EnableFraudDetection {
		return fmt.Errorf("mainnet requires fraud detection to be enabled")
	}

	if security.MaxTransactionAmount == "" {
		return fmt.Errorf("mainnet requires max_transaction_amount to be set")
	}

	if security.DailyVolumeLimit == "" {
		return fmt.Errorf("mainnet requires daily_volume_limit to be set")
	}

	return nil
}

// GetChainConfig returns the configuration for a specific chain
func (c *Config) GetChainConfig(chainName string) (*types.ChainConfig, error) {
	for i := range c.Chains {
		if c.Chains[i].Name == chainName {
			return &c.Chains[i], nil
		}
	}
	return nil, fmt.Errorf("chain config not found: %s", chainName)
}

// GetEVMChains returns all EVM chain configurations
func (c *Config) GetEVMChains() []types.ChainConfig {
	var evmChains []types.ChainConfig
	for _, chain := range c.Chains {
		if chain.ChainType == types.ChainTypeEVM {
			evmChains = append(evmChains, chain)
		}
	}
	return evmChains
}

// GetNonEVMChains returns all non-EVM chain configurations
func (c *Config) GetNonEVMChains() []types.ChainConfig {
	var nonEVMChains []types.ChainConfig
	for _, chain := range c.Chains {
		if chain.ChainType != types.ChainTypeEVM {
			nonEVMChains = append(nonEVMChains, chain)
		}
	}
	return nonEVMChains
}
