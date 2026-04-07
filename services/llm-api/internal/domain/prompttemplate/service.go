package prompttemplate

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"jan-server/services/llm-api/internal/domain/query"
	"jan-server/services/llm-api/internal/utils/idgen"
	"jan-server/services/llm-api/internal/utils/platformerrors"
)

// Service provides business logic for prompt template operations
type Service struct {
	repo PromptTemplateRepository
}

// NewService creates a new prompt template service
func NewService(repo PromptTemplateRepository) *Service {
	return &Service{repo: repo}
}

// GetByKey retrieves a prompt template by its unique template key
func (s *Service) GetByKey(ctx context.Context, templateKey string) (*PromptTemplate, error) {
	if templateKey == "" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "template key is required", nil, "c7a8b9d0-1234-5678-9abc-def012345678")
	}

	template, err := s.repo.FindByTemplateKey(ctx, templateKey)
	if err != nil {
		return nil, err
	}

	return template, nil
}

// GetByPublicID retrieves a prompt template by its public ID
func (s *Service) GetByPublicID(ctx context.Context, publicID string) (*PromptTemplate, error) {
	if publicID == "" {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "public ID is required", nil, "d8b9c0e1-2345-6789-abcd-ef0123456789")
	}

	template, err := s.repo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}

	return template, nil
}

// GetActive retrieves all active prompt templates
func (s *Service) GetActive(ctx context.Context) ([]*PromptTemplate, error) {
	isActive := true
	filter := PromptTemplateFilter{
		IsActive: &isActive,
	}

	templates, err := s.repo.FindByFilter(ctx, filter, nil)
	if err != nil {
		return nil, err
	}

	return templates, nil
}

// List retrieves prompt templates based on filter and pagination
func (s *Service) List(ctx context.Context, filter PromptTemplateFilter, p *query.Pagination) ([]*PromptTemplate, int64, error) {
	templates, err := s.repo.FindByFilter(ctx, filter, p)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.Count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return templates, count, nil
}

// Create creates a new prompt template
func (s *Service) Create(ctx context.Context, req CreatePromptTemplateRequest, createdBy *string) (*PromptTemplate, error) {
	// Check if template key already exists
	existing, err := s.repo.FindByTemplateKey(ctx, req.TemplateKey)
	if err != nil && !platformerrors.IsErrorType(err, platformerrors.ErrorTypeNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeConflict, "template key already exists", nil, "e9c0d1f2-3456-789a-bcde-f01234567890")
	}

	publicID, err := idgen.GenerateSecureID("pt", 24)
	if err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeInternal, "failed to generate ID", err, "e9c0d1f2-3456-789a-bcde-f01234567891")
	}

	template := &PromptTemplate{
		PublicID:    publicID,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		TemplateKey: req.TemplateKey,
		Content:     req.Content,
		Variables:   req.Variables,
		Metadata:    req.Metadata,
		IsActive:    true,
		IsSystem:    false,
		Version:     1,
		CreatedBy:   createdBy,
		UpdatedBy:   createdBy,
	}

	if err := s.repo.Create(ctx, template); err != nil {
		return nil, err
	}

	return template, nil
}

// Update updates an existing prompt template
func (s *Service) Update(ctx context.Context, publicID string, req UpdatePromptTemplateRequest, updatedBy *string) (*PromptTemplate, error) {
	template, err := s.repo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
		template.Name = *req.Name
	}
	if req.Description != nil {
		template.Description = req.Description
	}
	if req.Category != nil {
		template.Category = *req.Category
	}
	if req.Content != nil {
		template.Content = *req.Content
	}
	if req.Variables != nil {
		template.Variables = req.Variables
	}
	if req.Metadata != nil {
		template.Metadata = req.Metadata
	}
	if req.IsActive != nil {
		template.IsActive = *req.IsActive
	}

	template.Version++
	template.UpdatedBy = updatedBy

	if err := s.repo.Update(ctx, template); err != nil {
		return nil, err
	}

	return template, nil
}

// Delete deletes a prompt template (only non-system templates)
func (s *Service) Delete(ctx context.Context, publicID string) error {
	template, err := s.repo.FindByPublicID(ctx, publicID)
	if err != nil {
		return err
	}

	if template.IsSystem {
		return platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeForbidden, "system templates cannot be deleted", nil, "f0d1e2f3-4567-89ab-cdef-012345678901")
	}

	return s.repo.Delete(ctx, template.ID)
}

// Duplicate creates a copy of an existing prompt template
// It keeps the same template_key and only adds suffix to the name
func (s *Service) Duplicate(ctx context.Context, publicID string, newName string, createdBy *string) (*PromptTemplate, error) {
	original, err := s.repo.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}

	newPublicID, err := idgen.GenerateSecureID("pt", 24)
	if err != nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeInternal, "failed to generate ID", err, "01e2f3a4-5678-9abc-def0-123456789013")
	}

	// Use provided name or generate one
	duplicateName := newName
	if duplicateName == "" {
		duplicateName = fmt.Sprintf("%s - Copy", original.Name)
	}

	// Create duplicate - keep the same template_key
	duplicate := &PromptTemplate{
		PublicID:    newPublicID,
		Name:        duplicateName,
		Description: original.Description,
		Category:    original.Category,
		TemplateKey: original.TemplateKey, // Keep the same template_key
		Content:     original.Content,
		Variables:   original.Variables,
		Metadata:    original.Metadata,
		IsActive:    true,
		IsSystem:    false, // Duplicates are never system templates
		Version:     1,
		CreatedBy:   createdBy,
		UpdatedBy:   createdBy,
	}

	if err := s.repo.Create(ctx, duplicate); err != nil {
		return nil, err
	}

	return duplicate, nil
}

// RenderTemplate renders a prompt template with the given variables
func (s *Service) RenderTemplate(ctx context.Context, templateKey string, variables map[string]any) (string, error) {
	promptTemplate, err := s.repo.FindByTemplateKey(ctx, templateKey)
	if err != nil {
		return "", err
	}

	if !promptTemplate.IsActive {
		return "", platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "template is not active", nil, "12f3a4b5-6789-abcd-ef01-234567890123")
	}

	// If no variables, return content as-is
	if len(variables) == 0 {
		return promptTemplate.Content, nil
	}

	// Parse and execute template
	tmpl, err := template.New("prompt").Parse(promptTemplate.Content)
	if err != nil {
		return "", platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeInternal, "failed to parse template", err, "23a4b5c6-789a-bcde-f012-345678901234")
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, variables); err != nil {
		return "", platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeInternal, "failed to execute template", err, "34b5c6d7-89ab-cdef-0123-456789012345")
	}

	return result.String(), nil
}

// GetDeepResearchPrompt retrieves the Deep Research prompt template.
// If the template doesn't exist in the database, it returns a default template.
func (s *Service) GetDeepResearchPrompt(ctx context.Context) (*PromptTemplate, error) {
	template, err := s.GetByKey(ctx, TemplateKeyDeepResearch)
	if err != nil {
		// If not found, return the default template
		if platformerrors.IsErrorType(err, platformerrors.ErrorTypeNotFound) {
			return s.getDefaultDeepResearchTemplate(), nil
		}
		return nil, err
	}
	return template, nil
}

// getDefaultDeepResearchTemplate returns the hardcoded default Deep Research template
func (s *Service) getDefaultDeepResearchTemplate() *PromptTemplate {
	description := "Senior Research Agent prompt for conducting in-depth investigations with tool usage. Uses a 2-step workflow: clarification questions followed by comprehensive research and reporting."
	return &PromptTemplate{
		PublicID:    "pt_deep_research_default",
		Name:        "Deep Research Agent",
		Description: &description,
		Category:    CategoryOrchestration,
		TemplateKey: TemplateKeyDeepResearch,
		Content:     DefaultDeepResearchPrompt,
		Variables:   []string{},
		Metadata:    DefaultDeepResearchMetadata,
		IsActive:    true,
		IsSystem:    true,
		Version:     1,
	}
}

func (s *Service) getDefaultTimingTemplate() *PromptTemplate {
	description := "Provides current date context to the AI assistant"
	return &PromptTemplate{
		Name:        "Timing and Date Context",
		Description: &description,
		Category:    CategorySystem,
		TemplateKey: TemplateKeyTiming,
		Content:     DefaultTimingPrompt,
		Variables:   []string{"CurrentDate"},
		Metadata:    map[string]any{},
		IsActive:    true,
		IsSystem:    true,
		Version:     1,
	}
}

func (s *Service) getDefaultMemoryTemplate() *PromptTemplate {
	description := "Injects user-specific memory and preferences into prompts"
	return &PromptTemplate{
		Name:        "User Memory Injection",
		Description: &description,
		Category:    CategoryOrchestration,
		TemplateKey: TemplateKeyMemory,
		Content:     DefaultMemoryPrompt,
		Variables:   []string{"MemoryItems"},
		Metadata:    map[string]any{},
		IsActive:    true,
		IsSystem:    true,
		Version:     1,
	}
}

func (s *Service) getDefaultToolInstructionsTemplate() *PromptTemplate {
	description := "Provides guidance for using available tools effectively"
	return &PromptTemplate{
		Name:        "Tool Usage Instructions",
		Description: &description,
		Category:    CategoryTool,
		TemplateKey: TemplateKeyToolInstructions,
		Content:     DefaultToolInstructionsPrompt,
		Variables:   []string{},
		Metadata:    map[string]any{},
		IsActive:    true,
		IsSystem:    true,
		Version:     1,
	}
}

func (s *Service) getDefaultCodeAssistantTemplate() *PromptTemplate {
	description := "Provides guidelines for code assistance and programming help"
	return &PromptTemplate{
		Name:        "Code Assistant Instructions",
		Description: &description,
		Category:    CategoryOrchestration,
		TemplateKey: TemplateKeyCodeAssistant,
		Content:     DefaultCodeAssistantPrompt,
		Variables:   []string{},
		Metadata:    map[string]any{},
		IsActive:    true,
		IsSystem:    true,
		Version:     1,
	}
}

func (s *Service) getDefaultChainOfThoughtTemplate() *PromptTemplate {
	description := "Encourages step-by-step reasoning for complex questions"
	return &PromptTemplate{
		Name:        "Chain of Thought Reasoning",
		Description: &description,
		Category:    CategoryReasoning,
		TemplateKey: TemplateKeyChainOfThought,
		Content:     DefaultChainOfThoughtPrompt,
		Variables:   []string{},
		Metadata:    map[string]any{},
		IsActive:    true,
		IsSystem:    true,
		Version:     1,
	}
}

func (s *Service) getDefaultUserProfileTemplate() *PromptTemplate {
	description := "Injects user profile preferences and context for personalized responses"
	return &PromptTemplate{
		Name:        "User Profile Personalization",
		Description: &description,
		Category:    CategoryOrchestration,
		TemplateKey: TemplateKeyUserProfile,
		Content:     DefaultUserProfilePrompt,
		Variables:   []string{"BaseStyle", "BaseStyleInstruction", "CustomInstructions", "UserContext"},
		Metadata:    map[string]any{},
		IsActive:    true,
		IsSystem:    true,
		Version:     1,
	}
}

// EnsureDefaultTemplates seeds the database with default system templates if they don't exist.
// This should be called during application startup.
func (s *Service) EnsureDefaultTemplates(ctx context.Context) error {
	templates := map[string]func() *PromptTemplate{
		TemplateKeyDeepResearch:     s.getDefaultDeepResearchTemplate,
		TemplateKeyTiming:           s.getDefaultTimingTemplate,
		TemplateKeyMemory:           s.getDefaultMemoryTemplate,
		TemplateKeyToolInstructions: s.getDefaultToolInstructionsTemplate,
		TemplateKeyCodeAssistant:    s.getDefaultCodeAssistantTemplate,
		TemplateKeyChainOfThought:   s.getDefaultChainOfThoughtTemplate,
		TemplateKeyUserProfile:      s.getDefaultUserProfileTemplate,
	}

	for templateKey, getTemplate := range templates {
		_, err := s.repo.FindByTemplateKey(ctx, templateKey)
		if err != nil {
			if platformerrors.IsErrorType(err, platformerrors.ErrorTypeNotFound) {
				// Create the default template
				defaultTemplate := getTemplate()
				publicID, idErr := idgen.GenerateSecureID("pt", 24)
				if idErr != nil {
					return fmt.Errorf("failed to generate ID for %s template: %w", templateKey, idErr)
				}
				defaultTemplate.PublicID = publicID
				if createErr := s.repo.Create(ctx, defaultTemplate); createErr != nil {
					return fmt.Errorf("failed to create default %s template: %w", templateKey, createErr)
				}
			} else {
				return fmt.Errorf("failed to check for %s template: %w", templateKey, err)
			}
		}
	}
	return nil
}
