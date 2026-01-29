package modelprompttemplate

import (
	"context"

	"github.com/rs/zerolog/log"

	"jan-server/mono/apps/backend/internal/domain/prompttemplate"
	"jan-server/mono/apps/backend/internal/utils/platformerrors"
)

// Service provides business logic for model prompt template operations
type Service struct {
	repo               ModelPromptTemplateRepository
	promptTemplateRepo prompttemplate.PromptTemplateRepository
}

// NewService creates a new model prompt template service
func NewService(
	repo ModelPromptTemplateRepository,
	promptTemplateRepo prompttemplate.PromptTemplateRepository,
) *Service {
	return &Service{
		repo:               repo,
		promptTemplateRepo: promptTemplateRepo,
	}
}

// AssignTemplate assigns a prompt template to a model for a specific template key
// If an assignment already exists, it will be updated
func (s *Service) AssignTemplate(
	ctx context.Context,
	modelCatalogID string,
	req AssignTemplateRequest,
	userID *string,
) (*ModelPromptTemplate, error) {
	// Validate prompt template exists and get its UUID
	promptTemplate, err := s.promptTemplateRepo.FindByPublicID(ctx, req.PromptTemplateID)
	if err != nil {
		if platformerrors.IsErrorType(err, platformerrors.ErrorTypeNotFound) {
			return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeNotFound,
				"prompt template not found", nil, "mpt-svc-assign-001")
		}
		return nil, err
	}

	// Use the actual UUID from the prompt template for the foreign key
	promptTemplateUUID := promptTemplate.ID

	// Check if assignment already exists
	existing, err := s.repo.FindByModelAndKey(ctx, modelCatalogID, req.TemplateKey)
	if err != nil && !platformerrors.IsErrorType(err, platformerrors.ErrorTypeNotFound) {
		return nil, err
	}

	if existing != nil {
		// Update existing assignment
		existing.PromptTemplateID = promptTemplateUUID
		if req.Priority != nil {
			existing.Priority = *req.Priority
		}
		if req.IsActive != nil {
			existing.IsActive = *req.IsActive
		}
		existing.UpdatedBy = userID

		if err := s.repo.Update(ctx, existing); err != nil {
			return nil, err
		}

		// Attach the prompt template to the response
		existing.PromptTemplate = promptTemplate

		log.Debug().
			Str("model_catalog_id", modelCatalogID).
			Str("template_key", req.TemplateKey).
			Str("prompt_template_id", promptTemplate.PublicID).
			Msg("Updated model prompt template assignment")

		return existing, nil
	}

	// Create new assignment
	priority := 0
	if req.Priority != nil {
		priority = *req.Priority
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	mpt := &ModelPromptTemplate{
		ModelCatalogID:   modelCatalogID,
		TemplateKey:      req.TemplateKey,
		PromptTemplateID: promptTemplateUUID,
		Priority:         priority,
		IsActive:         isActive,
		CreatedBy:        userID,
		UpdatedBy:        userID,
	}

	if err := s.repo.Create(ctx, mpt); err != nil {
		return nil, err
	}

	// Attach the prompt template to the response
	mpt.PromptTemplate = promptTemplate

	log.Debug().
		Str("model_catalog_id", modelCatalogID).
		Str("template_key", req.TemplateKey).
		Str("prompt_template_id", promptTemplate.PublicID).
		Msg("Created model prompt template assignment")

	return mpt, nil
}

// UnassignTemplate removes a prompt template assignment from a model
func (s *Service) UnassignTemplate(ctx context.Context, modelCatalogID, templateKey string) error {
	if err := s.repo.Delete(ctx, modelCatalogID, templateKey); err != nil {
		return err
	}

	log.Debug().
		Str("model_catalog_id", modelCatalogID).
		Str("template_key", templateKey).
		Msg("Removed model prompt template assignment")

	return nil
}

// ListTemplatesForModel lists all template assignments for a model
func (s *Service) ListTemplatesForModel(ctx context.Context, modelCatalogID string) ([]*ModelPromptTemplate, error) {
	return s.repo.FindByModelWithTemplates(ctx, modelCatalogID)
}

// GetAssignment gets a specific template assignment for a model
func (s *Service) GetAssignment(ctx context.Context, modelCatalogID, templateKey string) (*ModelPromptTemplate, error) {
	return s.repo.FindByModelAndKey(ctx, modelCatalogID, templateKey)
}

// GetEffectiveTemplates returns a map of template keys to their resolved templates for a model
// This includes both model-specific assignments and global defaults
func (s *Service) GetEffectiveTemplates(ctx context.Context, modelCatalogID string) (*EffectiveTemplatesResponse, error) {
	response := &EffectiveTemplatesResponse{
		Templates: make(map[string]EffectiveTemplate),
	}

	// Get all model-specific assignments with templates
	assignments, err := s.repo.FindByModelWithTemplates(ctx, modelCatalogID)
	if err != nil {
		return nil, err
	}

	// Add model-specific templates
	for _, assignment := range assignments {
		if assignment.IsActive && assignment.PromptTemplate != nil {
			response.Templates[assignment.TemplateKey] = EffectiveTemplate{
				Template: assignment.PromptTemplate,
				Source:   "model_specific",
			}
		}
	}

	// Get all known template keys and fill in with global defaults
	templateKeys := []string{
		prompttemplate.TemplateKeyDeepResearch,
		prompttemplate.TemplateKeyTiming,
		prompttemplate.TemplateKeyMemory,
		prompttemplate.TemplateKeyToolInstructions,
		prompttemplate.TemplateKeyCodeAssistant,
		prompttemplate.TemplateKeyChainOfThought,
		prompttemplate.TemplateKeyUserProfile,
	}

	for _, key := range templateKeys {
		if _, exists := response.Templates[key]; !exists {
			// Try to get global template
			template, err := s.promptTemplateRepo.FindByTemplateKey(ctx, key)
			if err == nil && template != nil && template.IsActive {
				response.Templates[key] = EffectiveTemplate{
					Template: template,
					Source:   "global_default",
				}
			}
		}
	}

	return response, nil
}

// GetTemplateForModelByKey resolves the template for a specific key and model
// Priority: Model-specific → Global default → nil
func (s *Service) GetTemplateForModelByKey(
	ctx context.Context,
	modelCatalogID string,
	templateKey string,
) (*prompttemplate.PromptTemplate, string, error) {
	log.Debug().
		Str("model_catalog_id", modelCatalogID).
		Str("template_key", templateKey).
		Msg("Resolving prompt template for model")

	// 1. Try model-specific template
	if modelCatalogID != "" {
		assignment, err := s.repo.FindByModelAndKey(ctx, modelCatalogID, templateKey)
		if err == nil && assignment != nil && assignment.IsActive {
			log.Debug().
				Str("model_catalog_id", modelCatalogID).
				Str("template_key", templateKey).
				Str("prompt_template_uuid", assignment.PromptTemplateID).
				Bool("assignment_active", assignment.IsActive).
				Msg("Found model-specific template assignment")

			// Fetch the actual prompt template using UUID (not public_id)
			template, err := s.promptTemplateRepo.FindByID(ctx, assignment.PromptTemplateID)
			if err == nil && template != nil && template.IsActive {
				log.Debug().
					Str("model_catalog_id", modelCatalogID).
					Str("template_key", templateKey).
					Str("template_public_id", template.PublicID).
					Str("template_name", template.Name).
					Str("source", "model_specific").
					Int("content_length", len(template.Content)).
					Msg("Using model-specific template override")
				return template, "model_specific", nil
			} else if err != nil {
				log.Warn().
					Err(err).
					Str("model_catalog_id", modelCatalogID).
					Str("template_key", templateKey).
					Str("prompt_template_uuid", assignment.PromptTemplateID).
					Msg("Failed to fetch assigned template, falling back to global")
			} else if template != nil && !template.IsActive {
				log.Debug().
					Str("model_catalog_id", modelCatalogID).
					Str("template_key", templateKey).
					Str("template_public_id", template.PublicID).
					Msg("Model-specific template is inactive, falling back to global")
			}
		} else if err != nil && !platformerrors.IsErrorType(err, platformerrors.ErrorTypeNotFound) {
			log.Warn().
				Err(err).
				Str("model_catalog_id", modelCatalogID).
				Str("template_key", templateKey).
				Msg("Error finding model-specific template assignment")
		} else {
			log.Debug().
				Str("model_catalog_id", modelCatalogID).
				Str("template_key", templateKey).
				Msg("No model-specific template assignment found")
		}
	}

	// 2. Fall back to global default
	template, err := s.promptTemplateRepo.FindByTemplateKey(ctx, templateKey)
	if err == nil && template != nil && template.IsActive {
		log.Debug().
			Str("model_catalog_id", modelCatalogID).
			Str("template_key", templateKey).
			Str("template_public_id", template.PublicID).
			Str("template_name", template.Name).
			Str("source", "global_default").
			Int("content_length", len(template.Content)).
			Msg("Using global default template")
		return template, "global_default", nil
	}

	// 3. Return nil (caller should use hardcoded fallback)
	log.Debug().
		Str("model_catalog_id", modelCatalogID).
		Str("template_key", templateKey).
		Msg("No template found, caller should use hardcoded fallback")
	return nil, "", err
}

// UpdateAssignment updates an existing template assignment
func (s *Service) UpdateAssignment(
	ctx context.Context,
	modelCatalogID, templateKey string,
	req UpdateAssignmentRequest,
	userID *string,
) (*ModelPromptTemplate, error) {
	existing, err := s.repo.FindByModelAndKey(ctx, modelCatalogID, templateKey)
	if err != nil {
		return nil, err
	}

	if req.PromptTemplateID != nil {
		// Validate new template exists
		_, err := s.promptTemplateRepo.FindByPublicID(ctx, *req.PromptTemplateID)
		if err != nil {
			if platformerrors.IsErrorType(err, platformerrors.ErrorTypeNotFound) {
				return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeNotFound,
					"prompt template not found", nil, "mpt-svc-update-001")
			}
			return nil, err
		}
		existing.PromptTemplateID = *req.PromptTemplateID
	}

	if req.Priority != nil {
		existing.Priority = *req.Priority
	}

	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	existing.UpdatedBy = userID

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}

	log.Debug().
		Str("model_catalog_id", modelCatalogID).
		Str("template_key", templateKey).
		Msg("Updated model prompt template assignment")

	return existing, nil
}
