package project

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"jan-server/services/llm-api/internal/utils/idgen"
)

// ===============================================
// Project Validation
// ===============================================

// ProjectValidationConfig holds project-level validation rules
type ProjectValidationConfig struct {
	MaxNameLength        int
	MaxInstructionLength int
}

// DefaultProjectValidationConfig returns default project validation rules
func DefaultProjectValidationConfig() *ProjectValidationConfig {
	return &ProjectValidationConfig{
		MaxNameLength:        120,
		MaxInstructionLength: 32768, // 32k chars
	}
}

// ProjectValidator handles project-level validation
type ProjectValidator struct {
	config             *ProjectValidationConfig
	invalidCharPattern *regexp.Regexp
}

// NewProjectValidator creates a validator for projects
func NewProjectValidator(config *ProjectValidationConfig) *ProjectValidator {
	if config == nil {
		config = DefaultProjectValidationConfig()
	}

	// Pattern to detect control characters (except newline, tab, carriage return)
	invalidCharPattern := regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`)

	return &ProjectValidator{
		config:             config,
		invalidCharPattern: invalidCharPattern,
	}
}

// ValidateProject performs full project validation
func (v *ProjectValidator) ValidateProject(proj *Project) error {
	if proj == nil {
		return fmt.Errorf("project cannot be nil")
	}

	// Validate PublicID format
	if proj.PublicID != "" {
		if err := v.ValidateProjectID(proj.PublicID); err != nil {
			return fmt.Errorf("invalid project ID: %w", err)
		}
	}

	// Validate name
	if err := v.validateName(proj.Name); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	// Validate instruction
	if proj.Instruction != nil {
		if err := v.validateInstruction(*proj.Instruction); err != nil {
			return fmt.Errorf("invalid instruction: %w", err)
		}
	}

	return nil
}

// ValidateProjectID validates project ID format
func (v *ProjectValidator) ValidateProjectID(id string) error {
	if id == "" {
		return fmt.Errorf("project ID cannot be empty")
	}

	// Must start with "proj_" prefix
	if !strings.HasPrefix(id, "proj_") {
		return fmt.Errorf("project ID must start with 'proj_' prefix")
	}

	// Use domain-specific ID validation
	if !idgen.ValidateIDFormat(id, "proj") {
		return fmt.Errorf("invalid project ID format")
	}

	return nil
}

// validateName validates project name (internal use only)
func (v *ProjectValidator) validateName(name string) error {
	// Trim whitespace for validation
	trimmedName := strings.TrimSpace(name)

	if trimmedName == "" {
		return fmt.Errorf("name cannot be empty or only whitespace")
	}

	// Check length
	if utf8.RuneCountInString(trimmedName) > v.config.MaxNameLength {
		return fmt.Errorf("name exceeds maximum length of %d characters", v.config.MaxNameLength)
	}

	// Check for control characters
	if v.invalidCharPattern.MatchString(trimmedName) {
		return fmt.Errorf("name contains invalid control characters")
	}

	// Check for unprintable characters
	for _, r := range trimmedName {
		if !unicode.IsPrint(r) && r != '\n' && r != '\t' && r != '\r' {
			return fmt.Errorf("name contains unprintable characters")
		}
	}

	return nil
}

// validateInstruction validates instruction text (internal use only)
func (v *ProjectValidator) validateInstruction(instruction string) error {
	// Trim whitespace for validation
	trimmedInstruction := strings.TrimSpace(instruction)

	if trimmedInstruction == "" {
		// Empty instruction is allowed (optional field)
		return nil
	}

	// Check length
	if utf8.RuneCountInString(trimmedInstruction) > v.config.MaxInstructionLength {
		return fmt.Errorf("instruction exceeds maximum length of %d characters", v.config.MaxInstructionLength)
	}

	// Check for control characters (except newline, tab, carriage return which are allowed in text)
	for _, r := range trimmedInstruction {
		if !unicode.IsPrint(r) && r != '\n' && r != '\t' && r != '\r' {
			return fmt.Errorf("instruction contains unprintable characters")
		}
	}

	return nil
}

// ValidateProjectName validates project name independently
func (v *ProjectValidator) ValidateProjectName(name string) error {
	return v.validateName(name)
}

// ValidateProjectInstruction validates instruction independently
func (v *ProjectValidator) ValidateProjectInstruction(instruction string) error {
	return v.validateInstruction(instruction)
}
