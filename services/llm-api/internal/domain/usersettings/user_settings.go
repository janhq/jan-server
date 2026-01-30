// Package usersettings provides domain models for user preferences and settings.
package usersettings

import (
	"context"
	"encoding/json"
	"time"

	"jan-server/services/llm-api/internal/config"
)

// UserSettings represents user preferences and feature toggles.
type UserSettings struct {
	ID     uint
	UserID uint

	// Memory Configuration stored as JSON
	MemoryConfig MemoryConfig `gorm:"type:jsonb;serializer:json"`

	// Profile Settings
	ProfileSettings ProfileSettings `gorm:"type:jsonb;serializer:json"`

	// Advanced Settings
	AdvancedSettings AdvancedSettings `gorm:"type:jsonb;serializer:json"`

	// Other Feature Toggles
	EnableTrace bool
	EnableTools bool

	// Preferences - flexible JSON for future extensions
	Preferences map[string]interface{}

	CreatedAt time.Time
	UpdatedAt time.Time
}

// MemoryConfig returns memory configuration derived from settings.
type MemoryConfig struct {
	Enabled          bool    `json:"enabled"`
	ObserveEnabled   bool    `json:"observe_enabled"`
	InjectUserCore   bool    `json:"inject_user_core"`
	InjectSemantic   bool    `json:"inject_semantic"`
	InjectEpisodic   bool    `json:"inject_episodic"`
	MaxUserItems     int     `json:"max_user_items"`
	MaxProjectItems  int     `json:"max_project_items"`
	MaxEpisodicItems int     `json:"max_episodic_items"`
	MinSimilarity    float32 `json:"min_similarity"`
}

// BaseStyle represents the conversation style preference.
type BaseStyle string

const (
	BaseStyleConcise      BaseStyle = "Concise"
	BaseStyleFriendly     BaseStyle = "Friendly"
	BaseStyleProfessional BaseStyle = "Professional"
)

// IsValid checks if the base style is one of the allowed values.
func (bs BaseStyle) IsValid() bool {
	return bs == BaseStyleConcise || bs == BaseStyleFriendly || bs == BaseStyleProfessional
}

// ProfileSettings stores user profile information.
type ProfileSettings struct {
	BaseStyle          BaseStyle `json:"base_style"`          // Conversation style: Concise, Friendly, or Professional
	CustomInstructions string    `json:"custom_instructions"` // Additional behavior, style, and tone preferences
	NickName           string    `json:"nick_name"`           // What should Jan call you? (alias: nickname)
	Occupation         string    `json:"occupation"`          // User's occupation
	MoreAboutYou       string    `json:"more_about_you"`      // Additional information about the user
}

// AdvancedSettings stores advanced feature toggles.
type AdvancedSettings struct {
	WebSearch   bool `json:"web_search"`   // Let Jan automatically search the web for answers
	CodeEnabled bool `json:"code_enabled"` // Enable code execution features
}

// DefaultMemoryConfig returns default memory configuration
func DefaultMemoryConfig() MemoryConfig {
	return MemoryConfig{
		Enabled:          true,
		ObserveEnabled:   true, // Default ON - auto-learn from conversations
		InjectUserCore:   true,
		InjectSemantic:   true,
		InjectEpisodic:   false,
		MaxUserItems:     3,
		MaxProjectItems:  5,
		MaxEpisodicItems: 3,
		MinSimilarity:    0.75,
	}
}

// DefaultProfileSettings returns default profile settings
func DefaultProfileSettings() ProfileSettings {
	return ProfileSettings{
		BaseStyle:          BaseStyleFriendly, // Default to Friendly style
		CustomInstructions: "",
		NickName:           "",
		Occupation:         "",
		MoreAboutYou:       "",
	}
}

// DefaultAdvancedSettings returns default advanced settings
func DefaultAdvancedSettings() AdvancedSettings {
	return AdvancedSettings{
		WebSearch:   false, // Default OFF for privacy
		CodeEnabled: false, // Default OFF for security
	}
}

// DefaultPreferences returns default preference values.
func DefaultPreferences() map[string]interface{} {
	return map[string]interface{}{
		"selected_model":       "", // Will be set to first model from /v1/models
		"enable_deep_research": false,
		"enable_thinking":      false,
		"enable_search":        true,
		"enable_browser":       false,
		"hide_connectors":      true,
		"hide_artifacts":       true,
	}
}

// DefaultUserSettings returns settings with safe defaults.
func DefaultUserSettings(userID uint) *UserSettings {
	return &UserSettings{
		UserID:           userID,
		MemoryConfig:     DefaultMemoryConfig(),
		ProfileSettings:  DefaultProfileSettings(),
		AdvancedSettings: DefaultAdvancedSettings(),
		EnableTrace:      false,
		EnableTools:      true,
		Preferences:      DefaultPreferences(),
	}
}

// UpdateRequest represents fields that can be updated via API.
type UpdateRequest struct {
	MemoryConfig     *MemoryConfig          `json:"memory_config,omitempty"`
	ProfileSettings  *ProfileSettings       `json:"profile_settings,omitempty"`
	AdvancedSettings *AdvancedSettings      `json:"advanced_settings,omitempty"`
	EnableTrace      *bool                  `json:"enable_trace,omitempty"`
	EnableTools      *bool                  `json:"enable_tools,omitempty"`
	Preferences      map[string]interface{} `json:"preferences,omitempty"`
}

// Apply updates the UserSettings with non-nil fields from UpdateRequest.
func (s *UserSettings) Apply(req UpdateRequest) {
	if req.MemoryConfig != nil {
		s.MemoryConfig = *req.MemoryConfig
	}
	if req.ProfileSettings != nil {
		s.ProfileSettings = *req.ProfileSettings
	}
	if req.AdvancedSettings != nil {
		s.AdvancedSettings = *req.AdvancedSettings
	}
	if req.EnableTrace != nil {
		s.EnableTrace = *req.EnableTrace
	}
	if req.EnableTools != nil {
		s.EnableTools = *req.EnableTools
	}
	if req.Preferences != nil {
		// Merge preferences instead of replacing them
		if s.Preferences == nil {
			s.Preferences = DefaultPreferences()
		}
		for key, value := range req.Preferences {
			s.Preferences[key] = value
		}
	}
}

type profileSettingsAlias ProfileSettings

// MarshalJSON ensures we consistently emit nick_name while keeping the struct lean.
func (p ProfileSettings) MarshalJSON() ([]byte, error) {
	return json.Marshal(profileSettingsAlias(p))
}

// UnmarshalJSON accepts both nick_name and the legacy nickname key for backward compatibility.
func (p *ProfileSettings) UnmarshalJSON(data []byte) error {
	var aux struct {
		profileSettingsAlias
		NicknameLegacy string `json:"nickname"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	*p = ProfileSettings(aux.profileSettingsAlias)
	if p.NickName == "" && aux.NicknameLegacy != "" {
		p.NickName = aux.NicknameLegacy
	}

	return nil
}

// Repository defines storage operations for user settings.
type Repository interface {
	FindByUserID(ctx context.Context, userID uint) (*UserSettings, error)
	Upsert(ctx context.Context, settings *UserSettings) (*UserSettings, error)
	Update(ctx context.Context, settings *UserSettings) error
}

// ModelProvider provides access to model information for default settings.
type ModelProvider interface {
	GetFirstActiveModelID(ctx context.Context) (string, error)
}

// Service manages user settings operations.
type Service struct {
	repo          Repository
	modelProvider ModelProvider
	cfg           *config.Config
}

// NewService constructs a Service with required dependencies.
func NewService(repo Repository, modelProvider ModelProvider, cfg *config.Config) *Service {
	return &Service{
		repo:          repo,
		modelProvider: modelProvider,
		cfg:           cfg,
	}
}

func (s *Service) defaultPreferences() map[string]interface{} {
	defaults := DefaultPreferences()
	if s == nil || s.cfg == nil {
		return defaults
	}

	defaults["hide_connectors"] = s.cfg.PreferencesDefaultHideConnectors
	defaults["hide_artifacts"] = s.cfg.PreferencesDefaultHideArtifacts

	return defaults
}

// GetOrCreateSettings retrieves existing settings or creates defaults for a user.
func (s *Service) GetOrCreateSettings(ctx context.Context, userID uint) (*UserSettings, error) {
	settings, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Create default settings if none exist
	if settings == nil {
		defaults := DefaultUserSettings(userID)
		defaults.Preferences = s.defaultPreferences()

		// Set default selected_model from first active model
		if s.modelProvider != nil {
			if modelID, err := s.modelProvider.GetFirstActiveModelID(ctx); err == nil && modelID != "" {
				defaults.Preferences["selected_model"] = modelID
			}
		}

		return s.repo.Upsert(ctx, defaults)
	}

	// Ensure preferences always have default values (for existing users with empty preferences)
	s.ensureDefaultPreferences(ctx, settings)

	return settings, nil
}

// ensureDefaultPreferences merges default preferences into existing settings.
// Missing keys are filled with defaults; existing keys are preserved.
func (s *Service) ensureDefaultPreferences(ctx context.Context, settings *UserSettings) {
	if settings.Preferences == nil {
		settings.Preferences = make(map[string]interface{})
	}

	defaults := s.defaultPreferences()

	// Fill in missing preference keys with defaults
	for key, defaultValue := range defaults {
		if _, exists := settings.Preferences[key]; !exists {
			settings.Preferences[key] = defaultValue
		}
	}

	// Set selected_model from first active model if empty
	if selectedModel, ok := settings.Preferences["selected_model"].(string); !ok || selectedModel == "" {
		if s.modelProvider != nil {
			if modelID, err := s.modelProvider.GetFirstActiveModelID(ctx); err == nil && modelID != "" {
				settings.Preferences["selected_model"] = modelID
			}
		}
	}
}

// UpdateSettings applies updates to user settings.
func (s *Service) UpdateSettings(ctx context.Context, userID uint, req UpdateRequest) (*UserSettings, error) {
	settings, err := s.GetOrCreateSettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	settings.Apply(req)

	if err := s.repo.Update(ctx, settings); err != nil {
		return nil, err
	}

	return settings, nil
}
