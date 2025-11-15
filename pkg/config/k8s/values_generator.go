// Package k8s provides Kubernetes Helm values generation from configuration
package k8s

import (
	"fmt"
	"os"
	"strings"

	"github.com/janhq/jan-server/pkg/config"
	"gopkg.in/yaml.v3"
)

// ValuesGenerator generates Helm values.yaml from Config
type ValuesGenerator struct {
	config *config.Config
}

// NewValuesGenerator creates a new Helm values generator
func NewValuesGenerator(cfg *config.Config) *ValuesGenerator {
	return &ValuesGenerator{config: cfg}
}

// HelmValues represents the Helm chart values structure
type HelmValues struct {
	Global         GlobalValues             `yaml:"global"`
	Services       map[string]ServiceValues `yaml:"services"`
	Infrastructure InfrastructureValues     `yaml:"infrastructure"`
}

// GlobalValues contains global Helm chart settings
type GlobalValues struct {
	Environment     string            `yaml:"environment"`
	ImageRegistry   string            `yaml:"imageRegistry,omitempty"`
	ImagePullPolicy string            `yaml:"imagePullPolicy,omitempty"`
	Labels          map[string]string `yaml:"labels,omitempty"`
	Annotations     map[string]string `yaml:"annotations,omitempty"`
}

// ServiceValues contains service-specific Helm values
type ServiceValues struct {
	Enabled      bool              `yaml:"enabled"`
	ReplicaCount int               `yaml:"replicaCount,omitempty"`
	Image        ImageConfig       `yaml:"image,omitempty"`
	Service      ServiceConfig     `yaml:"service,omitempty"`
	Resources    ResourceConfig    `yaml:"resources,omitempty"`
	Env          map[string]string `yaml:"env,omitempty"`
	ConfigMap    map[string]string `yaml:"configMap,omitempty"`
	Secrets      []string          `yaml:"secrets,omitempty"`
	HealthChecks HealthCheckConfig `yaml:"healthChecks,omitempty"`
}

// ImageConfig contains Docker image configuration
type ImageConfig struct {
	Repository string `yaml:"repository"`
	Tag        string `yaml:"tag"`
	PullPolicy string `yaml:"pullPolicy,omitempty"`
}

// ServiceConfig contains Kubernetes service configuration
type ServiceConfig struct {
	Type       string `yaml:"type"`
	Port       int    `yaml:"port"`
	TargetPort int    `yaml:"targetPort,omitempty"`
	NodePort   int    `yaml:"nodePort,omitempty"`
}

// ResourceConfig contains resource limits and requests
type ResourceConfig struct {
	Limits   ResourceSpec `yaml:"limits,omitempty"`
	Requests ResourceSpec `yaml:"requests,omitempty"`
}

// ResourceSpec contains CPU and memory specs
type ResourceSpec struct {
	CPU    string `yaml:"cpu,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}

// HealthCheckConfig contains liveness and readiness probes
type HealthCheckConfig struct {
	LivenessProbe  ProbeConfig `yaml:"livenessProbe,omitempty"`
	ReadinessProbe ProbeConfig `yaml:"readinessProbe,omitempty"`
}

// ProbeConfig contains probe configuration
type ProbeConfig struct {
	HTTPGet             HTTPGetAction `yaml:"httpGet,omitempty"`
	InitialDelaySeconds int           `yaml:"initialDelaySeconds,omitempty"`
	PeriodSeconds       int           `yaml:"periodSeconds,omitempty"`
	TimeoutSeconds      int           `yaml:"timeoutSeconds,omitempty"`
	SuccessThreshold    int           `yaml:"successThreshold,omitempty"`
	FailureThreshold    int           `yaml:"failureThreshold,omitempty"`
}

// HTTPGetAction contains HTTP probe configuration
type HTTPGetAction struct {
	Path   string `yaml:"path"`
	Port   int    `yaml:"port"`
	Scheme string `yaml:"scheme,omitempty"`
}

// InfrastructureValues contains infrastructure component values
type InfrastructureValues struct {
	Database DatabaseValues `yaml:"database"`
	Auth     AuthValues     `yaml:"auth"`
}

// DatabaseValues contains database configuration for Helm
type DatabaseValues struct {
	Postgres PostgresValues `yaml:"postgres"`
}

// PostgresValues contains PostgreSQL Helm values
type PostgresValues struct {
	Enabled        bool              `yaml:"enabled"`
	Host           string            `yaml:"host"`
	Port           int               `yaml:"port"`
	Database       string            `yaml:"database"`
	User           string            `yaml:"user"`
	PasswordSecret string            `yaml:"passwordSecret"`
	SSLMode        string            `yaml:"sslMode"`
	MaxConnections int               `yaml:"maxConnections"`
	Resources      ResourceConfig    `yaml:"resources,omitempty"`
	Persistence    PersistenceConfig `yaml:"persistence,omitempty"`
}

// PersistenceConfig contains persistence configuration
type PersistenceConfig struct {
	Enabled      bool   `yaml:"enabled"`
	StorageClass string `yaml:"storageClass,omitempty"`
	Size         string `yaml:"size"`
}

// AuthValues contains authentication configuration for Helm
type AuthValues struct {
	Keycloak KeycloakValues `yaml:"keycloak"`
}

// KeycloakValues contains Keycloak Helm values
type KeycloakValues struct {
	Enabled        bool           `yaml:"enabled"`
	BaseURL        string         `yaml:"baseURL"`
	PublicURL      string         `yaml:"publicURL,omitempty"`
	AdminUser      string         `yaml:"adminUser"`
	AdminRealm     string         `yaml:"adminRealm"`
	PasswordSecret string         `yaml:"passwordSecret"`
	Resources      ResourceConfig `yaml:"resources,omitempty"`
}

// Generate creates Helm values from Config
func (g *ValuesGenerator) Generate() (*HelmValues, error) {
	values := &HelmValues{
		Global: GlobalValues{
			Environment:     g.config.Meta.Environment,
			ImagePullPolicy: "IfNotPresent",
			Labels: map[string]string{
				"app.kubernetes.io/name":        "jan-server",
				"app.kubernetes.io/version":     g.config.Meta.Version,
				"app.kubernetes.io/environment": g.config.Meta.Environment,
			},
		},
		Services:       make(map[string]ServiceValues),
		Infrastructure: g.generateInfrastructure(),
	}

	// Generate service values
	if err := g.generateServices(values); err != nil {
		return nil, err
	}

	return values, nil
}

// generateServices creates service-specific Helm values
func (g *ValuesGenerator) generateServices(values *HelmValues) error {
	// LLM API
	values.Services["llm-api"] = ServiceValues{
		Enabled:      true,
		ReplicaCount: 2,
		Image: ImageConfig{
			Repository: "jan-llm-api",
			Tag:        g.config.Meta.Version,
		},
		Service: ServiceConfig{
			Type:       "ClusterIP",
			Port:       g.config.Services.LLMAPI.HTTPPort,
			TargetPort: g.config.Services.LLMAPI.HTTPPort,
		},
		Resources: ResourceConfig{
			Limits: ResourceSpec{
				CPU:    "1000m",
				Memory: "1Gi",
			},
			Requests: ResourceSpec{
				CPU:    "500m",
				Memory: "512Mi",
			},
		},
		HealthChecks: HealthCheckConfig{
			LivenessProbe: ProbeConfig{
				HTTPGet: HTTPGetAction{
					Path: "/health",
					Port: g.config.Services.LLMAPI.HTTPPort,
				},
				InitialDelaySeconds: 30,
				PeriodSeconds:       10,
				TimeoutSeconds:      5,
				FailureThreshold:    3,
			},
			ReadinessProbe: ProbeConfig{
				HTTPGet: HTTPGetAction{
					Path: "/health",
					Port: g.config.Services.LLMAPI.HTTPPort,
				},
				InitialDelaySeconds: 10,
				PeriodSeconds:       5,
				TimeoutSeconds:      3,
				FailureThreshold:    3,
			},
		},
		ConfigMap: map[string]string{
			"LOG_LEVEL":  g.config.Services.LLMAPI.LogLevel,
			"LOG_FORMAT": g.config.Services.LLMAPI.LogFormat,
		},
		Secrets: []string{"database-credentials", "keycloak-credentials"},
	}

	// MCP Tools
	values.Services["mcp-tools"] = ServiceValues{
		Enabled:      true,
		ReplicaCount: 2,
		Image: ImageConfig{
			Repository: "jan-mcp-tools",
			Tag:        g.config.Meta.Version,
		},
		Service: ServiceConfig{
			Type:       "ClusterIP",
			Port:       g.config.Services.MCPTools.HTTPPort,
			TargetPort: g.config.Services.MCPTools.HTTPPort,
		},
		Resources: ResourceConfig{
			Limits: ResourceSpec{
				CPU:    "500m",
				Memory: "512Mi",
			},
			Requests: ResourceSpec{
				CPU:    "250m",
				Memory: "256Mi",
			},
		},
		HealthChecks: HealthCheckConfig{
			LivenessProbe: ProbeConfig{
				HTTPGet: HTTPGetAction{
					Path: "/health",
					Port: g.config.Services.MCPTools.HTTPPort,
				},
				InitialDelaySeconds: 20,
				PeriodSeconds:       10,
				TimeoutSeconds:      5,
				FailureThreshold:    3,
			},
			ReadinessProbe: ProbeConfig{
				HTTPGet: HTTPGetAction{
					Path: "/health",
					Port: g.config.Services.MCPTools.HTTPPort,
				},
				InitialDelaySeconds: 10,
				PeriodSeconds:       5,
				TimeoutSeconds:      3,
				FailureThreshold:    3,
			},
		},
		ConfigMap: map[string]string{
			"LOG_LEVEL":  g.config.Services.MCPTools.LogLevel,
			"LOG_FORMAT": g.config.Services.MCPTools.LogFormat,
		},
	}

	// Media API
	values.Services["media-api"] = ServiceValues{
		Enabled:      true,
		ReplicaCount: 2,
		Image: ImageConfig{
			Repository: "jan-media-api",
			Tag:        g.config.Meta.Version,
		},
		Service: ServiceConfig{
			Type:       "ClusterIP",
			Port:       g.config.Services.MediaAPI.HTTPPort,
			TargetPort: g.config.Services.MediaAPI.HTTPPort,
		},
		Resources: ResourceConfig{
			Limits: ResourceSpec{
				CPU:    "500m",
				Memory: "512Mi",
			},
			Requests: ResourceSpec{
				CPU:    "250m",
				Memory: "256Mi",
			},
		},
	}

	// Response API
	values.Services["response-api"] = ServiceValues{
		Enabled:      true,
		ReplicaCount: 2,
		Image: ImageConfig{
			Repository: "jan-response-api",
			Tag:        g.config.Meta.Version,
		},
		Service: ServiceConfig{
			Type:       "ClusterIP",
			Port:       g.config.Services.ResponseAPI.HTTPPort,
			TargetPort: g.config.Services.ResponseAPI.HTTPPort,
		},
		Resources: ResourceConfig{
			Limits: ResourceSpec{
				CPU:    "500m",
				Memory: "512Mi",
			},
			Requests: ResourceSpec{
				CPU:    "250m",
				Memory: "256Mi",
			},
		},
	}

	return nil
}

// generateInfrastructure creates infrastructure Helm values
func (g *ValuesGenerator) generateInfrastructure() InfrastructureValues {
	return InfrastructureValues{
		Database: DatabaseValues{
			Postgres: PostgresValues{
				Enabled:        true,
				Host:           g.config.Infrastructure.Database.Postgres.Host,
				Port:           g.config.Infrastructure.Database.Postgres.Port,
				Database:       g.config.Infrastructure.Database.Postgres.Database,
				User:           g.config.Infrastructure.Database.Postgres.User,
				PasswordSecret: "postgres-password",
				SSLMode:        g.config.Infrastructure.Database.Postgres.SSLMode,
				MaxConnections: g.config.Infrastructure.Database.Postgres.MaxConnections,
				Resources: ResourceConfig{
					Limits: ResourceSpec{
						CPU:    "2000m",
						Memory: "2Gi",
					},
					Requests: ResourceSpec{
						CPU:    "1000m",
						Memory: "1Gi",
					},
				},
				Persistence: PersistenceConfig{
					Enabled: true,
					Size:    "10Gi",
				},
			},
		},
		Auth: AuthValues{
			Keycloak: KeycloakValues{
				Enabled:        true,
				BaseURL:        g.config.Infrastructure.Auth.Keycloak.BaseURL,
				PublicURL:      g.config.Infrastructure.Auth.Keycloak.PublicURL,
				AdminUser:      g.config.Infrastructure.Auth.Keycloak.AdminUser,
				AdminRealm:     g.config.Infrastructure.Auth.Keycloak.AdminRealm,
				PasswordSecret: "keycloak-admin-password",
				Resources: ResourceConfig{
					Limits: ResourceSpec{
						CPU:    "1000m",
						Memory: "1Gi",
					},
					Requests: ResourceSpec{
						CPU:    "500m",
						Memory: "512Mi",
					},
				},
			},
		},
	}
}

// GenerateToFile writes Helm values to a file
func (g *ValuesGenerator) GenerateToFile(outputPath string) error {
	values, err := g.Generate()
	if err != nil {
		return fmt.Errorf("generate values: %w", err)
	}

	data, err := yaml.Marshal(values)
	if err != nil {
		return fmt.Errorf("marshal YAML: %w", err)
	}

	// Add header comment
	header := fmt.Sprintf("# Helm values generated from configuration\n# Environment: %s\n# Version: %s\n\n",
		g.config.Meta.Environment, g.config.Meta.Version)

	output := header + string(data)

	if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// GenerateToString returns Helm values as a YAML string
func (g *ValuesGenerator) GenerateToString() (string, error) {
	values, err := g.Generate()
	if err != nil {
		return "", fmt.Errorf("generate values: %w", err)
	}

	data, err := yaml.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("marshal YAML: %w", err)
	}

	header := fmt.Sprintf("# Helm values generated from configuration\n# Environment: %s\n# Version: %s\n\n",
		g.config.Meta.Environment, g.config.Meta.Version)

	return header + string(data), nil
}

// GenerateWithOverrides generates Helm values with environment-specific overrides
func (g *ValuesGenerator) GenerateWithOverrides(environment string) (*HelmValues, error) {
	values, err := g.Generate()
	if err != nil {
		return nil, err
	}

	// Apply environment-specific overrides
	switch strings.ToLower(environment) {
	case "production":
		g.applyProductionOverrides(values)
	case "staging":
		g.applyStagingOverrides(values)
	case "development":
		g.applyDevelopmentOverrides(values)
	}

	return values, nil
}

// applyProductionOverrides applies production-specific settings
func (g *ValuesGenerator) applyProductionOverrides(values *HelmValues) {
	values.Global.ImagePullPolicy = "Always"

	// Increase replica counts
	for name, svc := range values.Services {
		svc.ReplicaCount = 3
		values.Services[name] = svc
	}

	// Enable persistence
	values.Infrastructure.Database.Postgres.Persistence.Enabled = true
	values.Infrastructure.Database.Postgres.Persistence.Size = "50Gi"
}

// applyStagingOverrides applies staging-specific settings
func (g *ValuesGenerator) applyStagingOverrides(values *HelmValues) {
	values.Global.ImagePullPolicy = "IfNotPresent"

	// Moderate replica counts
	for name, svc := range values.Services {
		svc.ReplicaCount = 2
		values.Services[name] = svc
	}

	values.Infrastructure.Database.Postgres.Persistence.Enabled = true
	values.Infrastructure.Database.Postgres.Persistence.Size = "20Gi"
}

// applyDevelopmentOverrides applies development-specific settings
func (g *ValuesGenerator) applyDevelopmentOverrides(values *HelmValues) {
	values.Global.ImagePullPolicy = "Never"

	// Single replica for development
	for name, svc := range values.Services {
		svc.ReplicaCount = 1
		// Reduce resources for development
		svc.Resources.Limits.CPU = "500m"
		svc.Resources.Limits.Memory = "512Mi"
		svc.Resources.Requests.CPU = "100m"
		svc.Resources.Requests.Memory = "128Mi"
		values.Services[name] = svc
	}

	// Disable persistence in development
	values.Infrastructure.Database.Postgres.Persistence.Enabled = false
}
