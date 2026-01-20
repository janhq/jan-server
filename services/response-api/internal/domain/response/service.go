package response

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"jan-server/services/response-api/internal/domain/agent"
	"jan-server/services/response-api/internal/domain/conversation"
	"jan-server/services/response-api/internal/domain/llm"
	"jan-server/services/response-api/internal/domain/plan"
	"jan-server/services/response-api/internal/domain/status"
	"jan-server/services/response-api/internal/domain/tool"
	"jan-server/services/response-api/internal/infrastructure/media"
	"jan-server/services/response-api/internal/utils/idgen"
	"jan-server/services/response-api/internal/webhook"
)

// ServiceImpl provides the domain implementation.
type ServiceImpl struct {
	responses         Repository
	conversations     conversation.Repository
	conversationItems conversation.ItemRepository
	toolExecutions    ToolExecutionRepository
	orchestrator      *tool.Orchestrator
	agentOrchestrator agent.Orchestrator
	mcpClient         tool.MCPClient
	mediaClient       *media.Client
	modelInfoProvider llm.ModelInfoProvider
	webhookService    webhook.Service
	agentRegistry     agent.Registry
	planService       plan.Service
	log               zerolog.Logger
}

// NewService wires dependencies.
func NewService(
	responses Repository,
	conversations conversation.Repository,
	conversationItems conversation.ItemRepository,
	toolExecutions ToolExecutionRepository,
	orchestrator *tool.Orchestrator,
	agentOrchestrator agent.Orchestrator,
	mcpClient tool.MCPClient,
	mediaClient *media.Client,
	modelInfoProvider llm.ModelInfoProvider,
	webhookService webhook.Service,
	agentRegistry agent.Registry,
	planService plan.Service,
	log zerolog.Logger,
) *ServiceImpl {
	return &ServiceImpl{
		responses:         responses,
		conversations:     conversations,
		conversationItems: conversationItems,
		toolExecutions:    toolExecutions,
		orchestrator:      orchestrator,
		agentOrchestrator: agentOrchestrator,
		mcpClient:         mcpClient,
		mediaClient:       mediaClient,
		modelInfoProvider: modelInfoProvider,
		webhookService:    webhookService,
		agentRegistry:     agentRegistry,
		planService:       planService,
		log:               log.With().Str("component", "response-service").Logger(),
	}
}

// Create orchestrates a complete response lifecycle.
// If Background=true, it enqueues the task and returns immediately.
// Otherwise, it executes synchronously.
func (s *ServiceImpl) Create(ctx context.Context, params CreateParams) (*Response, error) {
	// Validate background constraints
	if params.Background && !params.Store {
		return nil, errors.New("background mode requires store=true")
	}

	if params.Background {
		return s.createAsync(ctx, params)
	}
	return s.createSync(ctx, params)
}

// createAsync creates a response record with status=queued and returns immediately.
func (s *ServiceImpl) createAsync(ctx context.Context, params CreateParams) (*Response, error) {
	var conv *conversation.Conversation
	var err error

	// Handle conversation context
	if params.PreviousResponseID != nil && strings.TrimSpace(*params.PreviousResponseID) != "" {
		prevResp, err := s.responses.FindByPublicID(ctx, *params.PreviousResponseID)
		if err != nil {
			s.log.Warn().Err(err).Str("previous_response_id", *params.PreviousResponseID).Msg("failed to load previous response")
		} else if prevResp.ConversationPublicID != nil {
			conv, err = s.conversations.FindByPublicID(ctx, *prevResp.ConversationPublicID)
			if err != nil {
				s.log.Warn().Err(err).Str("conversation_id", *prevResp.ConversationPublicID).Msg("failed to load conversation")
			}
		}
	}

	if conv == nil {
		if params.ConversationID != nil && strings.TrimSpace(*params.ConversationID) != "" {
			conv, err = s.conversations.FindByPublicID(ctx, *params.ConversationID)
			if err != nil {
				return nil, fmt.Errorf("fetch conversation: %w", err)
			}
		} else {
			conv = &conversation.Conversation{
				PublicID: newPublicID("conv"),
				UserID:   params.UserID,
			}
			if err := s.conversations.Create(ctx, conv); err != nil {
				return nil, fmt.Errorf("create conversation: %w", err)
			}
		}
	}

	now := time.Now()
	responseModel := &Response{
		PublicID:             newPublicID("resp"),
		Object:               "response",
		UserID:               params.UserID,
		Model:                params.Model,
		SystemPrompt:         params.SystemPrompt,
		Input:                params.Input,
		Status:               StatusQueued,
		Stream:               params.Stream,
		Background:           params.Background,
		Store:                params.Store,
		APIKey:               params.APIKey, // Store API key for background execution
		Metadata:             params.Metadata,
		ConversationID:       &conv.ID,
		ConversationPublicID: &conv.PublicID,
		PreviousResponseID:   params.PreviousResponseID,
		CreatedAt:            now,
		UpdatedAt:            now,
		QueuedAt:             &now,
	}

	if err := s.responses.Create(ctx, responseModel); err != nil {
		return nil, fmt.Errorf("create response: %w", err)
	}

	s.log.Info().
		Str("response_id", responseModel.PublicID).
		Str("user_id", params.UserID).
		Str("model", params.Model).
		Msg("background response queued")

	return responseModel, nil
}

// createSync executes the response synchronously (original behavior).
func (s *ServiceImpl) createSync(ctx context.Context, params CreateParams) (*Response, error) {
	var conv *conversation.Conversation
	var err error
	conversationID := ""

	// If PreviousResponseID is provided, load that response's conversation for context
	if params.PreviousResponseID != nil && strings.TrimSpace(*params.PreviousResponseID) != "" {
		prevResp, err := s.responses.FindByPublicID(ctx, *params.PreviousResponseID)
		if err != nil {
			s.log.Warn().Err(err).Str("previous_response_id", *params.PreviousResponseID).Msg("failed to load previous response, continuing without context")
		} else if prevResp.ConversationPublicID != nil {
			// Use the previous response's conversation to maintain context
			conv, err = s.conversations.FindByPublicID(ctx, *prevResp.ConversationPublicID)
			if err != nil {
				s.log.Warn().Err(err).Str("conversation_id", *prevResp.ConversationPublicID).Msg("failed to load conversation, creating new one")
			}
		}
	}

	if conv != nil {
		conversationID = conv.PublicID
	}

	// If we still don't have a conversation, check for explicit conversation_id or create new
	if conv == nil {
		if params.ConversationID != nil && strings.TrimSpace(*params.ConversationID) != "" {
			conv, err = s.conversations.FindByPublicID(ctx, *params.ConversationID)
			if err != nil {
				return nil, fmt.Errorf("fetch conversation: %w", err)
			}
		} else {
			conv = &conversation.Conversation{
				PublicID: newPublicID("conv"),
				UserID:   params.UserID,
			}
			if err := s.conversations.Create(ctx, conv); err != nil {
				return nil, fmt.Errorf("create conversation: %w", err)
			}
		}
	}

	existingItems, err := s.conversationItems.ListByConversationID(ctx, conv.ID)
	if err != nil {
		return nil, fmt.Errorf("list conversation items: %w", err)
	}

	responseModel := &Response{
		PublicID:             newPublicID("resp"),
		Object:               "response",
		UserID:               params.UserID,
		Model:                params.Model,
		SystemPrompt:         params.SystemPrompt,
		Input:                params.Input,
		Status:               StatusInProgress,
		Stream:               params.Stream,
		Background:           params.Background,
		Store:                params.Store,
		Metadata:             params.Metadata,
		ConversationID:       &conv.ID,
		ConversationPublicID: &conv.PublicID,
		PreviousResponseID:   params.PreviousResponseID,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	if params.StreamObserver != nil {
		params.StreamObserver.OnResponseCreated(responseModel)
	}

	if err := s.responses.Create(ctx, responseModel); err != nil {
		return nil, fmt.Errorf("create response: %w", err)
	}

	agentType := s.extractAgentType(responseModel.Metadata)
	if agentType != "" && s.agentRegistry != nil && s.planService != nil {
		if err := s.executeWithPlan(ctx, responseModel, agentType); err != nil {
			s.log.Warn().Err(err).Str("response_id", responseModel.PublicID).Msg("plan execution failed")
		}
		updated, err := s.responses.FindByPublicID(ctx, responseModel.PublicID)
		if err != nil {
			return nil, err
		}
		return updated, nil
	}

	baseMessages, err := s.buildBaseMessages(params.SystemPrompt, existingItems)
	if err != nil {
		return s.failResponse(ctx, responseModel, fmt.Errorf("build base messages: %w", err))
	}

	userMessages, convoItems, err := s.convertInputToMessages(conv.ID, len(existingItems), params.Input)
	if err != nil {
		return s.failResponse(ctx, responseModel, err)
	}
	messages := append(baseMessages, userMessages...)
	initialLength := len(messages)

	toolDefs := params.Tools
	if len(toolDefs) == 0 {
		if toolDefs, err = s.fetchAvailableTools(ctx); err != nil {
			return s.failResponse(ctx, responseModel, err)
		}
	}

	// Fetch model context length for message trimming
	var contextLength *int
	if s.modelInfoProvider != nil {
		modelInfo, err := s.modelInfoProvider.GetModelInfo(ctx, params.Model)
		if err != nil {
			s.log.Warn().Err(err).Str("model", params.Model).Msg("failed to fetch model info, using default context length")
		} else if modelInfo != nil && modelInfo.ContextLength != nil {
			contextLength = modelInfo.ContextLength
		}
	}

	execParams := func(defs []llm.ToolDefinition, toolChoice *llm.ToolChoice) tool.ExecuteParams {
		return tool.ExecuteParams{
			Ctx:             ctx,
			Model:           params.Model,
			Messages:        messages,
			RequestID:       params.RequestID,
			ConversationID:  conversationID,
			UserID:          params.UserID,
			Temperature:     params.Temperature,
			MaxTokens:       params.MaxTokens,
			ContextLength:   contextLength,
			ToolChoice:      toolChoice,
			ToolDefinitions: defs,
			StreamObserver:  params.StreamObserver,
		}
	}

	orchestratorResult, err := s.orchestrator.Execute(execParams(toolDefs, params.ToolChoice))
	if err != nil && shouldRetryWithoutTools(err) && len(toolDefs) > 0 {
		s.log.Warn().Err(err).Str("response_id", responseModel.PublicID).Msg("llm provider rejected tool definitions, retrying without tools")
		orchestratorResult, err = s.orchestrator.Execute(execParams(nil, nil))
	}
	if err != nil {
		return s.failResponse(ctx, responseModel, err)
	}

	responseModel.Status = StatusCompleted
	responseModel.Output = orchestratorResult.FinalMessage.Content
	responseModel.Usage = orchestratorResult.Usage
	now := time.Now()
	responseModel.CompletedAt = &now
	responseModel.UpdatedAt = now

	s.populateArtifactsAndCitations(ctx, responseModel, nil)

	if err := s.responses.Update(ctx, responseModel); err != nil {
		return nil, err
	}

	if err := s.toolExecutions.RecordExecutions(ctx, responseModel.ID, orchestratorResult.Executions); err != nil {
		s.log.Error().Err(err).Str("response_id", responseModel.PublicID).Msg("store tool executions failed")
	}

	newMessages := orchestratorResult.Messages[initialLength:]
	newItems := append(convoItems, s.convertMessagesToItems(conv.ID, len(existingItems)+len(convoItems), newMessages)...)
	if err := s.conversationItems.BulkInsert(ctx, newItems); err != nil {
		s.log.Error().Err(err).Str("response_id", responseModel.PublicID).Msg("store conversation items failed")
	}

	return responseModel, nil
}

// GetByPublicID returns the response by id.
func (s *ServiceImpl) GetByPublicID(ctx context.Context, publicID string) (*Response, error) {
	return s.responses.FindByPublicID(ctx, publicID)
}

// GetWithPlanDetails returns the response with its associated plan, tasks, and steps.
func (s *ServiceImpl) GetWithPlanDetails(ctx context.Context, publicID string) (*ResponseWithPlan, error) {
	resp, err := s.responses.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}

	result := &ResponseWithPlan{
		Response: resp,
	}

	// Try to get plan details if plan service is available
	if s.planService != nil {
		planDetails, err := s.planService.GetByResponseID(ctx, publicID)
		if err == nil && planDetails != nil {
			// Get full plan with tasks and steps
			fullPlan, err := s.planService.GetPlanWithDetails(ctx, planDetails.ID)
			if err == nil {
				result.PlanDetails = fullPlan
			} else {
				// Fallback to basic plan info
				result.PlanDetails = planDetails
			}
		}
		// If no plan exists, PlanDetails will be nil (omitempty)
	}

	return result, nil
}

// Cancel marks the response as cancelled.
// For queued tasks, this prevents them from being picked up by workers.
// For in-progress tasks, workers should periodically check cancellation status.
func (s *ServiceImpl) Cancel(ctx context.Context, publicID string) (*Response, error) {
	resp, err := s.responses.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}

	// Already in terminal state
	if resp.Status == StatusCompleted || resp.Status == StatusCancelled || resp.Status == StatusFailed {
		return resp, nil
	}

	// Cancel the response
	if err := s.responses.MarkCancelled(ctx, resp); err != nil {
		return nil, err
	}

	s.log.Info().
		Str("response_id", resp.PublicID).
		Str("previous_status", string(resp.Status)).
		Msg("response cancelled")

	return resp, nil
}

// Retry retries the last failed step for a response plan and resumes execution.
func (s *ServiceImpl) Retry(ctx context.Context, publicID string) (*Response, error) {
	if s.agentOrchestrator == nil {
		s.log.Error().Msg("retry requested but agent orchestrator not configured")
		return nil, errors.New("agent orchestrator not configured")
	}

	resp, err := s.responses.FindByPublicID(ctx, publicID)
	if err != nil {
		s.log.Error().Err(err).Str("response_id", publicID).Msg("retry failed to load response")
		return nil, err
	}

	planModel, err := s.planService.GetByResponseID(ctx, publicID)
	if err != nil {
		s.log.Error().Err(err).Str("response_id", publicID).Msg("retry failed to load plan")
		return nil, err
	}

	planDetails, err := s.planService.GetPlanWithDetails(ctx, planModel.ID)
	if err != nil {
		s.log.Error().Err(err).Str("plan_id", planModel.ID).Msg("retry failed to load plan details")
		return nil, err
	}

	if !planDetails.Status.IsRetryable() {
		s.log.Warn().
			Str("response_id", publicID).
			Str("plan_id", planDetails.ID).
			Str("plan_status", string(planDetails.Status)).
			Msg("retry rejected due to plan status")
		return nil, fmt.Errorf("plan status %q does not allow retry", planDetails.Status)
	}

	task, step := findFirstFailedStep(planDetails)
	if task == nil || step == nil {
		s.log.Warn().
			Str("response_id", publicID).
			Str("plan_id", planDetails.ID).
			Msg("retry requested but no failed steps found")
		return nil, errors.New("no failed steps to retry")
	}
	if !step.CanRetry() {
		s.log.Warn().
			Str("response_id", publicID).
			Str("plan_id", planDetails.ID).
			Str("step_id", step.ID).
			Int("retry_count", step.RetryCount).
			Int("max_retries", step.MaxRetries).
			Str("error_severity", string(step.ErrorSeverity)).
			Msg("retry rejected because step is not retryable")
		return nil, errors.New("step is not retryable")
	}

	if task.Status == status.StatusFailed || task.Status == status.StatusCompleted {
		if err := s.planService.ResetTaskForRetry(ctx, task.ID); err != nil {
			s.log.Error().Err(err).Str("task_id", task.ID).Msg("retry failed to reset task")
			return nil, err
		}
	}

	if task.TaskType == plan.TaskTypeFinalization && step.Sequence > 1 {
		if err := s.planService.ResetTaskStepsForRetry(ctx, task.ID, step.Sequence); err != nil {
			s.log.Error().Err(err).Str("task_id", task.ID).Msg("retry failed to reset task steps")
			return nil, err
		}
	}

	if _, err := s.planService.RetryStep(ctx, step.ID); err != nil {
		s.log.Error().Err(err).Str("step_id", step.ID).Msg("retry failed to reset step")
		return nil, err
	}

	if err := s.planService.UpdateStatus(ctx, planDetails.ID, status.StatusInProgress, nil); err != nil {
		s.log.Error().Err(err).Str("plan_id", planDetails.ID).Msg("retry failed to update plan status")
		return nil, err
	}

	now := time.Now()
	resp.Status = StatusInProgress
	resp.Error = nil
	resp.FailedAt = nil
	resp.CompletedAt = nil
	resp.UpdatedAt = now
	if resp.StartedAt == nil {
		resp.StartedAt = &now
	}
	if err := s.responses.Update(ctx, resp); err != nil {
		s.log.Error().Err(err).Str("response_id", publicID).Msg("retry failed to update response status")
		return nil, err
	}

	s.log.Info().
		Str("response_id", publicID).
		Str("plan_id", planDetails.ID).
		Str("task_id", task.ID).
		Str("step_id", step.ID).
		Msg("retrying plan from last failed step")

	if err := s.executePlanRetry(ctx, resp, planDetails); err != nil {
		s.log.Error().Err(err).Str("response_id", publicID).Str("plan_id", planDetails.ID).Msg("retry execution failed")
		return nil, err
	}

	return s.responses.FindByPublicID(ctx, publicID)
}

// ListConversationItems returns the textual conversation history for the response.
func (s *ServiceImpl) ListConversationItems(ctx context.Context, publicID string) ([]ConversationItem, error) {
	resp, err := s.responses.FindByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}

	if resp.ConversationID == nil {
		return nil, errors.New("response has no conversation")
	}

	items, err := s.conversationItems.ListByConversationID(ctx, *resp.ConversationID)
	if err != nil {
		return nil, err
	}

	result := make([]ConversationItem, 0, len(items))
	for _, item := range items {
		result = append(result, ConversationItem{
			Role:    string(item.GetRole()),
			Content: item.GetContent(),
			Status:  string(item.GetStatus()),
		})
	}
	return result, nil
}

func (s *ServiceImpl) failResponse(ctx context.Context, resp *Response, failure error) (*Response, error) {
	now := time.Now()
	resp.Status = StatusFailed
	resp.FailedAt = &now
	resp.Error = &ErrorDetails{
		Code:    "response_failed",
		Message: failure.Error(),
	}
	if err := s.responses.Update(ctx, resp); err != nil {
		s.log.Error().Err(err).Str("response_id", resp.PublicID).Msg("update failed response")
	}
	return nil, failure
}

func (s *ServiceImpl) executePlanRetry(ctx context.Context, resp *Response, planDetails *plan.Plan) error {
	maxIterations := planDetails.EstimatedSteps * 2
	if maxIterations <= 0 {
		maxIterations = 50
	}

	var lastOutput interface{}

	for i := 0; i < maxIterations; i++ {
		result, err := s.agentOrchestrator.ExecuteNextStep(ctx, planDetails.ID)
		if err != nil {
			now := time.Now()
			resp.Status = StatusFailed
			resp.Error = &ErrorDetails{Message: err.Error()}
			resp.CompletedAt = &now
			resp.UpdatedAt = now
			if updateErr := s.responses.Update(ctx, resp); updateErr != nil {
				s.log.Error().Err(updateErr).Msg("failed to update response after step failure")
			}
			return err
		}

		if result != nil && len(result.Output) > 0 {
			var decoded interface{}
			if unmarshalErr := json.Unmarshal(result.Output, &decoded); unmarshalErr == nil {
				lastOutput = decoded
			} else {
				lastOutput = string(result.Output)
			}
		}

		planStatus, statusErr := s.agentOrchestrator.GetStatus(ctx, planDetails.ID)
		if statusErr != nil {
			s.log.Warn().Err(statusErr).Str("plan_id", planDetails.ID).Msg("failed to get plan status")
			continue
		}

		switch planStatus.Status {
		case status.StatusCompleted:
			now := time.Now()
			resp.Status = StatusCompleted
			resp.CompletedAt = &now
			resp.UpdatedAt = now

			finalOutput := s.generatePlanOutput(ctx, planDetails.ID)
			if finalOutput == "" {
				if lastOutput != nil {
					finalOutput = lastOutput
				} else {
					finalOutput = "plan completed"
				}
			}
			resp.Output = finalOutput
			s.populateArtifactsAndCitations(ctx, resp, &planDetails.ID)
			if err := s.responses.Update(ctx, resp); err != nil {
				return fmt.Errorf("failed to update response: %w", err)
			}
			s.log.Info().
				Str("response_id", resp.PublicID).
				Str("plan_id", planDetails.ID).
				Msg("plan retry completed successfully")
			return nil
		case status.StatusFailed:
			now := time.Now()
			resp.Status = StatusFailed
			errMsg := "plan execution failed"
			if planStatus.LastError != nil {
				errMsg = *planStatus.LastError
			}
			resp.Error = &ErrorDetails{Message: errMsg}
			resp.CompletedAt = &now
			resp.UpdatedAt = now
			if err := s.responses.Update(ctx, resp); err != nil {
				return fmt.Errorf("failed to update response after failure: %w", err)
			}
			return fmt.Errorf("plan failed: %s", errMsg)
		case status.StatusWaitForUser:
			resp.Status = Status(status.StatusWaitForUser)
			resp.UpdatedAt = time.Now()
			if err := s.responses.Update(ctx, resp); err != nil {
				return fmt.Errorf("failed to update response while waiting for user: %w", err)
			}
			s.log.Info().
				Str("response_id", resp.PublicID).
				Str("plan_id", planDetails.ID).
				Msg("plan retry paused waiting for user input")
			return nil
		}
	}

	return fmt.Errorf("plan execution exceeded maximum iterations (%d)", maxIterations)
}

func findFirstFailedStep(planDetails *plan.Plan) (*plan.Task, *plan.Step) {
	if planDetails == nil {
		return nil, nil
	}

	var firstTask *plan.Task
	var firstStep *plan.Step
	firstTaskSeq := int(^uint(0) >> 1)
	firstStepSeq := int(^uint(0) >> 1)

	for tIdx := range planDetails.Tasks {
		task := &planDetails.Tasks[tIdx]
		for sIdx := range task.Steps {
			step := &task.Steps[sIdx]
			if step.Status != status.StatusFailed {
				continue
			}
			if task.Sequence < firstTaskSeq || (task.Sequence == firstTaskSeq && step.Sequence < firstStepSeq) {
				firstTask = task
				firstStep = step
				firstTaskSeq = task.Sequence
				firstStepSeq = step.Sequence
			}
		}
	}

	return firstTask, firstStep
}

func (s *ServiceImpl) buildBaseMessages(systemPrompt *string, items []conversation.Item) ([]llm.ChatMessage, error) {
	messages := make([]llm.ChatMessage, 0, len(items)+1)
	if systemPrompt != nil && strings.TrimSpace(*systemPrompt) != "" {
		messages = append(messages, llm.ChatMessage{
			Role:    "system",
			Content: strings.TrimSpace(*systemPrompt),
		})
	}

	for _, item := range items {
		messages = append(messages, llm.ChatMessage{
			Role:    string(item.GetRole()),
			Content: contentToLLM(item.GetContent()),
		})
	}
	return messages, nil
}

func (s *ServiceImpl) convertInputToMessages(conversationID uint, startingSeq int, input interface{}) ([]llm.ChatMessage, []conversation.Item, error) {
	var messages []llm.ChatMessage
	var convoItems []conversation.Item

	switch v := input.(type) {
	case string:
		msg := llm.ChatMessage{Role: "user", Content: strings.TrimSpace(v)}
		messages = append(messages, msg)
		convoItems = append(convoItems, newConversationItem(conversationID, startingSeq, msg))
	case []interface{}:
		for _, raw := range v {
			msg, err := mapToChatMessage(raw)
			if err != nil {
				return nil, nil, err
			}
			messages = append(messages, msg)
			convoItems = append(convoItems, newConversationItem(conversationID, startingSeq+len(convoItems), msg))
		}
	case map[string]interface{}:
		msg, err := mapToChatMessage(v)
		if err != nil {
			return nil, nil, err
		}
		messages = append(messages, msg)
		convoItems = append(convoItems, newConversationItem(conversationID, startingSeq, msg))
	default:
		bytes, _ := json.Marshal(input)
		msg := llm.ChatMessage{
			Role:    "user",
			Content: string(bytes),
		}
		messages = append(messages, msg)
		convoItems = append(convoItems, newConversationItem(conversationID, startingSeq, msg))
	}

	return messages, convoItems, nil
}

func (s *ServiceImpl) convertMessagesToItems(conversationID uint, startingSeq int, messages []llm.ChatMessage) []conversation.Item {
	items := make([]conversation.Item, 0, len(messages))
	for _, msg := range messages {
		items = append(items, newConversationItem(conversationID, startingSeq+len(items), msg))
	}
	return items
}

// convertMessagesToItemPtrs converts LLM messages to conversation item pointers.
// Using pointers allows BulkCreate to set the IDs back on the items.
func (s *ServiceImpl) convertMessagesToItemPtrs(conversationID uint, startingSeq int, messages []llm.ChatMessage) []*conversation.Item {
	items := make([]*conversation.Item, 0, len(messages))
	for i, msg := range messages {
		item := newConversationItem(conversationID, startingSeq+i, msg)
		items = append(items, &item)
	}
	return items
}

// conversationItemTracker tracks conversation items and their associated tool calls.
type conversationItemTracker struct {
	items           []*conversation.Item
	toolCallIDToIdx map[string]int   // Maps tool call ID to item index
	roleToIdxList   map[string][]int // Maps role to list of item indices
}

// buildItemTracker creates a tracker from conversation items and messages.
func buildItemTracker(items []*conversation.Item, messages []llm.ChatMessage) *conversationItemTracker {
	tracker := &conversationItemTracker{
		items:           items,
		toolCallIDToIdx: make(map[string]int),
		roleToIdxList:   make(map[string][]int),
	}

	for i, msg := range messages {
		if i >= len(items) {
			break
		}
		// Track by role
		tracker.roleToIdxList[msg.Role] = append(tracker.roleToIdxList[msg.Role], i)

		// Track tool result messages by tool_call_id
		if msg.ToolCallID != nil && *msg.ToolCallID != "" {
			tracker.toolCallIDToIdx[*msg.ToolCallID] = i
		}

		// Track assistant messages with tool calls
		for _, tc := range msg.ToolCalls {
			// The tool call itself is in an assistant message
			// We'll track this for linking
			if tc.ID != "" {
				// Store as negative index to differentiate from tool results
				// Actually, let's use a separate map or different approach
			}
		}
	}

	return tracker
}

// getItemByToolCallID returns the conversation item for a tool result.
func (t *conversationItemTracker) getItemByToolCallID(callID string) *conversation.Item {
	if idx, ok := t.toolCallIDToIdx[callID]; ok && idx < len(t.items) {
		return t.items[idx]
	}
	return nil
}

// getItemsByRole returns all items with the given role.
func (t *conversationItemTracker) getItemsByRole(role string) []*conversation.Item {
	result := make([]*conversation.Item, 0)
	if indices, ok := t.roleToIdxList[role]; ok {
		for _, idx := range indices {
			if idx < len(t.items) {
				result = append(result, t.items[idx])
			}
		}
	}
	return result
}

func (s *ServiceImpl) fetchAvailableTools(ctx context.Context) ([]llm.ToolDefinition, error) {
	mcpTools, err := s.mcpClient.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("list MCP tools: %w", err)
	}

	defs := make([]llm.ToolDefinition, 0, len(mcpTools))
	for _, tool := range mcpTools {
		defs = append(defs, tool.ToLLMTool())
	}
	return defs, nil
}

func newConversationItem(conversationID uint, sequence int, msg llm.ChatMessage) conversation.Item {
	legacyContent := normalizeContent(msg.Content)
	role := conversation.ItemRole(msg.Role)
	if role == "" {
		role = conversation.RoleUser
	}

	// Build proper Content array for database persistence
	var content []conversation.Content
	var itemType conversation.ItemType

	// Handle tool calls (assistant messages with tool calls)
	if len(msg.ToolCalls) > 0 {
		itemType = conversation.ItemTypeFunctionCall
		for _, tc := range msg.ToolCalls {
			argsStr := string(tc.Function.Arguments)
			content = append(content, conversation.Content{
				Type: "function_call",
				FunctionCall: &conversation.FunctionCall{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: argsStr,
				},
			})
		}
	} else if msg.ToolCallID != nil && *msg.ToolCallID != "" {
		// Tool result message
		itemType = conversation.ItemTypeFunctionCallOut
		textContent := extractTextContent(msg.Content)
		content = []conversation.Content{
			{
				Type: "function_call_output",
				FunctionCallOut: &conversation.FunctionCallOut{
					CallID: *msg.ToolCallID,
					Output: textContent,
				},
			},
		}
	} else {
		// Regular message (user/assistant/system)
		itemType = conversation.ItemTypeMessage
		textContent := extractTextContent(msg.Content)
		if textContent != "" {
			content = []conversation.Content{
				{
					Type:       "output_text",
					OutputText: &conversation.OutputText{Text: textContent},
				},
			}
		}
	}

	return conversation.Item{
		ConversationID: conversationID,
		Type:           itemType,
		Role:           role,
		Status:         conversation.ItemStatusCompleted,
		LegacyContent:  legacyContent,
		Content:        content,
		Sequence:       sequence + 1,
		SequenceNumber: sequence + 1,
		CreatedAt:      time.Now(),
	}
}

// extractTextContent extracts text from various content formats
func extractTextContent(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case map[string]interface{}:
		if text, ok := v["text"].(string); ok {
			return text
		}
	case map[string]string:
		if text, ok := v["text"]; ok {
			return text
		}
	case []interface{}:
		// Handle array of content items
		var parts []string
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if text, ok := m["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	return ""
}

func contentToLLM(content map[string]interface{}) interface{} {
	if content == nil {
		return nil
	}
	if text, ok := content["text"]; ok {
		return text
	}
	return content
}

func normalizeContent(content interface{}) map[string]interface{} {
	switch v := content.(type) {
	case string:
		return map[string]interface{}{"type": "text", "text": v}
	case map[string]interface{}:
		return v
	case []interface{}:
		return map[string]interface{}{"type": "list", "items": v}
	default:
		bytes, _ := json.Marshal(v)
		return map[string]interface{}{"type": "json", "text": string(bytes)}
	}
}

func mapToChatMessage(messageData interface{}) (llm.ChatMessage, error) {
	payload, ok := messageData.(map[string]interface{})
	if !ok {
		return llm.ChatMessage{}, errors.New("input items must be objects with role/content")
	}

	role, _ := payload["role"].(string)
	if role == "" {
		role = "user"
	}
	content := payload["content"]
	if content == nil {
		content = payload["text"]
	}
	if content == nil {
		return llm.ChatMessage{}, errors.New("input item missing content")
	}

	return llm.ChatMessage{
		Role:    role,
		Content: content,
	}, nil
}

func newPublicID(prefix string) string {
	// Use secure ID generation matching llm-api format (prefix_alphanumeric16)
	id, err := idgen.GenerateSecureID(prefix, 16)
	if err != nil {
		// Fallback should never happen with crypto/rand, but log and use basic fallback
		return idgen.MustGenerateSecureID(prefix, 16)
	}
	return id
}

func shouldRetryWithoutTools(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "failed to complete chat request") {
		return true
	}
	if strings.Contains(message, "tools unsupported") {
		return true
	}
	return false
}

// ExecuteBackground processes a queued background task.
// This method is called by workers from the worker pool.
func (s *ServiceImpl) ExecuteBackground(ctx context.Context, publicID string) error {
	// Load the response record
	resp, err := s.responses.FindByPublicID(ctx, publicID)
	if err != nil {
		return fmt.Errorf("failed to load response: %w", err)
	}
	requestID := publicID // reuse response identifier for traceability

	// Verify it's in a processable state
	if resp.Status != StatusInProgress {
		return fmt.Errorf("response %s is not in_progress (current: %s)", publicID, resp.Status)
	}

	// Inject API key into context for LLM API calls
	if resp.APIKey != nil && *resp.APIKey != "" {
		ctx = llm.ContextWithAuthToken(ctx, *resp.APIKey)
	}

	// Check if this is an agent-based request that requires a plan
	agentType := s.extractAgentType(resp.Metadata)
	if agentType != "" && s.agentRegistry != nil && s.planService != nil {
		s.log.Info().
			Str("response_id", publicID).
			Str("agent_type", agentType).
			Msg("detected agent request, creating plan")
		return s.executeWithPlan(ctx, resp, agentType)
	}

	// Load conversation for context
	if resp.ConversationID == nil {
		return errors.New("response has no conversation")
	}
	conv, err := s.conversations.FindByPublicID(ctx, *resp.ConversationPublicID)
	if err != nil {
		return fmt.Errorf("failed to load conversation: %w", err)
	}
	conversationID := conv.PublicID

	// Load conversation items (history)
	existingItems, err := s.conversationItems.ListByConversationID(ctx, conv.ID)
	if err != nil {
		return fmt.Errorf("failed to load conversation items: %w", err)
	}

	// Build messages from history and current input
	baseMessages, err := s.buildBaseMessages(resp.SystemPrompt, existingItems)
	if err != nil {
		return fmt.Errorf("build base messages: %w", err)
	}

	userMessages, convoItems, err := s.convertInputToMessages(conv.ID, len(existingItems), resp.Input)
	if err != nil {
		return fmt.Errorf("convert input: %w", err)
	}
	messages := append(baseMessages, userMessages...)
	initialLength := len(messages)

	// Load tool definitions
	toolDefs, err := s.fetchAvailableTools(ctx)
	if err != nil {
		s.log.Warn().Err(err).Msg("Failed to load MCP tools, continuing without tools")
		toolDefs = []llm.ToolDefinition{}
	}

	// Fetch model context length for message trimming
	var contextLength *int
	if s.modelInfoProvider != nil {
		modelInfo, err := s.modelInfoProvider.GetModelInfo(ctx, resp.Model)
		if err != nil {
			s.log.Warn().Err(err).Str("model", resp.Model).Msg("failed to fetch model info for background task, using default context length")
		} else if modelInfo != nil && modelInfo.ContextLength != nil {
			contextLength = modelInfo.ContextLength
		}
	}

	// Execute orchestration (no streaming in background mode)
	execParams := tool.ExecuteParams{
		Ctx:             ctx,
		Model:           resp.Model,
		Messages:        messages,
		RequestID:       requestID,
		ConversationID:  conversationID,
		UserID:          resp.UserID,
		Temperature:     nil, // Use model defaults for background tasks
		MaxTokens:       nil,
		ContextLength:   contextLength,
		ToolDefinitions: toolDefs,
		StreamObserver:  nil, // Background mode never streams
	}

	orchestratorResult, execErr := s.orchestrator.Execute(execParams)
	if execErr != nil && shouldRetryWithoutTools(execErr) && len(toolDefs) > 0 {
		s.log.Warn().Err(execErr).Str("response_id", resp.PublicID).Msg("llm provider rejected tool definitions, retrying without tools")
		execParams.ToolDefinitions = nil
		orchestratorResult, execErr = s.orchestrator.Execute(execParams)
	}

	// Update response status
	now := time.Now()
	if execErr != nil {
		resp.Status = StatusFailed
		resp.Error = &ErrorDetails{Message: execErr.Error()}
		resp.CompletedAt = &now
		resp.UpdatedAt = now
	} else {
		resp.Status = StatusCompleted
		resp.Output = orchestratorResult.FinalMessage.Content
		resp.Usage = orchestratorResult.Usage
		resp.CompletedAt = &now
		resp.UpdatedAt = now

		s.populateArtifactsAndCitations(ctx, resp, nil)

		// Record tool executions
		if err := s.toolExecutions.RecordExecutions(ctx, resp.ID, orchestratorResult.Executions); err != nil {
			s.log.Error().Err(err).Str("response_id", resp.PublicID).Msg("store tool executions failed")
		}

		// Record conversation items (skip initial messages, only store new ones)
		newMessages := orchestratorResult.Messages[initialLength:]
		newItems := append(convoItems, s.convertMessagesToItems(conv.ID, len(existingItems)+len(convoItems), newMessages)...)
		if err := s.conversationItems.BulkInsert(ctx, newItems); err != nil {
			s.log.Error().Err(err).Str("response_id", resp.PublicID).Msg("store conversation items failed")
		}
	}

	// Persist final state
	if err := s.responses.Update(ctx, resp); err != nil {
		return fmt.Errorf("failed to update response: %w", err)
	}

	// Send webhook notifications (async, don't block on webhook failures)
	go func() {
		webhookCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if execErr != nil {
			errorCode := "execution_failed"
			errorMsg := execErr.Error()
			if resp.Error != nil {
				errorCode = resp.Error.Code
				errorMsg = resp.Error.Message
			}
			if err := s.webhookService.NotifyFailed(webhookCtx, resp.PublicID, errorCode, errorMsg, resp.Metadata); err != nil {
				s.log.Error().Err(err).Str("response_id", resp.PublicID).Msg("webhook notification failed")
			}
		} else {
			if err := s.webhookService.NotifyCompleted(webhookCtx, resp.PublicID, resp.Output, resp.Metadata, resp.CompletedAt); err != nil {
				s.log.Error().Err(err).Str("response_id", resp.PublicID).Msg("webhook notification failed")
			}
		}
	}()

	return execErr
}

// extractAgentType extracts the agent_type from metadata if present.
func (s *ServiceImpl) extractAgentType(metadata map[string]interface{}) string {
	if metadata == nil {
		return ""
	}
	agentType, ok := metadata["agent_type"]
	if !ok {
		return ""
	}
	agentTypeStr, ok := agentType.(string)
	if !ok {
		return ""
	}
	return agentTypeStr
}

// executeWithPlan handles agent-based execution with plan creation and tracking.
func (s *ServiceImpl) executeWithPlan(ctx context.Context, resp *Response, agentType string) error {
	// Load conversation for context
	if resp.ConversationID == nil {
		return errors.New("response has no conversation")
	}
	conv, err := s.conversations.FindByPublicID(ctx, *resp.ConversationPublicID)
	if err != nil {
		return fmt.Errorf("failed to load conversation: %w", err)
	}

	// Load conversation items (history)
	existingItems, err := s.conversationItems.ListByConversationID(ctx, conv.ID)
	if err != nil {
		return fmt.Errorf("failed to load conversation items: %w", err)
	}

	// Extract user message from input
	userMessage := s.extractUserMessage(resp.Input)

	// Load tool definitions for the plan request
	toolDefs, err := s.fetchAvailableTools(ctx)
	if err != nil {
		s.log.Warn().Err(err).Msg("Failed to load MCP tools for plan, continuing without tools")
		toolDefs = []llm.ToolDefinition{}
	}

	// Convert tool definitions to agent format
	agentTools := make([]agent.ToolDefinition, len(toolDefs))
	for i, td := range toolDefs {
		paramsBytes, _ := json.Marshal(td.Function.Parameters)
		agentTools[i] = agent.ToolDefinition{
			Name:        td.Function.Name,
			Description: td.Function.Description,
			Parameters:  paramsBytes,
		}
	}

	// Convert history to agent format
	agentHistory := make([]agent.ConversationItem, len(existingItems))
	for i, item := range existingItems {
		content := ""
		// Convert Content slice to string
		if len(item.Content) > 0 {
			// Try to extract text from the first content item
			if b, err := json.Marshal(item.Content); err == nil {
				content = string(b)
			}
		}
		agentHistory[i] = agent.ConversationItem{
			Role:    string(item.Role),
			Content: content,
		}
	}

	// Build plan request
	planRequest := &agent.PlanRequest{
		ResponseID:          resp.PublicID,
		ConversationID:      conv.PublicID,
		UserMessage:         userMessage,
		SystemPrompt:        resp.SystemPrompt,
		Model:               resp.Model,
		Tools:               agentTools,
		ConversationHistory: agentHistory,
		Metadata:            resp.Metadata,
	}

	// Find a planner that can handle this request
	planner, err := s.agentRegistry.FindPlanner(ctx, planRequest)
	if err != nil {
		s.log.Warn().
			Err(err).
			Str("agent_type", agentType).
			Msg("no planner found for agent type, falling back to standard execution")
		// Fall through to standard execution by returning a marker error
		return s.executeWithoutPlan(ctx, resp)
	}

	s.log.Info().
		Str("response_id", resp.PublicID).
		Str("planner", planner.Name()).
		Msg("creating execution plan")

	// Create the plan
	planResult, err := planner.CreatePlan(ctx, planRequest)
	if err != nil {
		now := time.Now()
		resp.Status = StatusFailed
		resp.Error = &ErrorDetails{
			Code:    "plan_creation_failed",
			Message: fmt.Sprintf("failed to create plan: %v", err),
		}
		resp.CompletedAt = &now
		resp.UpdatedAt = now
		if updateErr := s.responses.Update(ctx, resp); updateErr != nil {
			s.log.Error().Err(updateErr).Msg("failed to update response after plan creation failure")
		}
		return err
	}

	s.log.Info().
		Str("response_id", resp.PublicID).
		Str("plan_id", planResult.Plan.ID).
		Int("task_count", len(planResult.Tasks)).
		Msg("plan created successfully")

	// Execute the plan using the standard orchestrator but track progress via plan
	return s.executePlanWithOrchestrator(ctx, resp, planResult)
}

// executeWithoutPlan falls back to standard tool orchestration without plan tracking.
func (s *ServiceImpl) executeWithoutPlan(ctx context.Context, resp *Response) error {
	// This is essentially the original ExecuteBackground logic
	// Load conversation for context
	if resp.ConversationID == nil {
		return errors.New("response has no conversation")
	}
	conv, err := s.conversations.FindByPublicID(ctx, *resp.ConversationPublicID)
	if err != nil {
		return fmt.Errorf("failed to load conversation: %w", err)
	}
	conversationID := conv.PublicID
	requestID := resp.PublicID

	// Load conversation items (history)
	existingItems, err := s.conversationItems.ListByConversationID(ctx, conv.ID)
	if err != nil {
		return fmt.Errorf("failed to load conversation items: %w", err)
	}

	// Build messages from history and current input
	baseMessages, err := s.buildBaseMessages(resp.SystemPrompt, existingItems)
	if err != nil {
		return fmt.Errorf("build base messages: %w", err)
	}

	userMessages, convoItems, err := s.convertInputToMessages(conv.ID, len(existingItems), resp.Input)
	if err != nil {
		return fmt.Errorf("convert input: %w", err)
	}
	messages := append(baseMessages, userMessages...)
	initialLength := len(messages)

	// Load tool definitions
	toolDefs, err := s.fetchAvailableTools(ctx)
	if err != nil {
		s.log.Warn().Err(err).Msg("Failed to load MCP tools, continuing without tools")
		toolDefs = []llm.ToolDefinition{}
	}

	// Execute orchestration
	execParams := tool.ExecuteParams{
		Ctx:             ctx,
		Model:           resp.Model,
		Messages:        messages,
		RequestID:       requestID,
		ConversationID:  conversationID,
		UserID:          resp.UserID,
		ToolDefinitions: toolDefs,
		StreamObserver:  nil,
	}

	orchestratorResult, execErr := s.orchestrator.Execute(execParams)
	if execErr != nil && shouldRetryWithoutTools(execErr) && len(toolDefs) > 0 {
		execParams.ToolDefinitions = nil
		orchestratorResult, execErr = s.orchestrator.Execute(execParams)
	}

	// Update response status
	now := time.Now()
	if execErr != nil {
		resp.Status = StatusFailed
		resp.Error = &ErrorDetails{Message: execErr.Error()}
		resp.CompletedAt = &now
		resp.UpdatedAt = now
	} else {
		resp.Status = StatusCompleted
		resp.Output = orchestratorResult.FinalMessage.Content
		resp.Usage = orchestratorResult.Usage
		resp.CompletedAt = &now
		resp.UpdatedAt = now

		s.populateArtifactsAndCitations(ctx, resp, nil)

		if err := s.toolExecutions.RecordExecutions(ctx, resp.ID, orchestratorResult.Executions); err != nil {
			s.log.Error().Err(err).Str("response_id", resp.PublicID).Msg("store tool executions failed")
		}

		newMessages := orchestratorResult.Messages[initialLength:]
		newItems := append(convoItems, s.convertMessagesToItems(conv.ID, len(existingItems)+len(convoItems), newMessages)...)
		if err := s.conversationItems.BulkInsert(ctx, newItems); err != nil {
			s.log.Error().Err(err).Str("response_id", resp.PublicID).Msg("store conversation items failed")
		}
	}

	if err := s.responses.Update(ctx, resp); err != nil {
		return fmt.Errorf("failed to update response: %w", err)
	}

	return execErr
}

func (s *ServiceImpl) populateArtifactsAndCitations(ctx context.Context, resp *Response, planID *string) {
	if resp == nil {
		return
	}

	if planID != nil && s.planService != nil {
		planDetails, err := s.planService.GetPlanWithDetails(ctx, *planID)
		if err != nil {
			s.log.Warn().Err(err).Str("plan_id", *planID).Msg("failed to load plan details for citations")
		} else if planDetails != nil {
			resp.Citations = planDetails.ExtractCitations()

			// Extract artifacts from plan and merge with existing
			planArtifacts := planDetails.ExtractArtifacts()
			if len(planArtifacts) > 0 {
				seenArtifacts := make(map[string]bool)
				for _, a := range resp.Artifacts {
					seenArtifacts[a.ID] = true
				}
				for _, a := range planArtifacts {
					if !seenArtifacts[a.ID] {
						resp.Artifacts = append(resp.Artifacts, a)
						seenArtifacts[a.ID] = true
					}
				}
			}
		}
	}

	if s.mediaClient == nil {
		return
	}

	outputText, ok := resp.Output.(string)
	if !ok || strings.TrimSpace(outputText) == "" {
		return
	}

	imagePaths := extractLocalImagePaths(outputText)
	if len(imagePaths) == 0 {
		return
	}

	seenArtifacts := make(map[string]bool)
	for _, artifact := range resp.Artifacts {
		seenArtifacts[artifact.ID] = true
	}

	updatedOutput := outputText
	for _, path := range imagePaths {
		artifact, err := s.uploadSandboxImage(ctx, resp, path)
		if err != nil {
			s.log.Warn().
				Err(err).
				Str("path", path).
				Str("response_id", resp.PublicID).
				Msg("failed to upload sandbox image")
			continue
		}

		if !seenArtifacts[artifact.ID] {
			resp.Artifacts = append(resp.Artifacts, *artifact)
			seenArtifacts[artifact.ID] = true
		}

		updatedOutput = strings.ReplaceAll(updatedOutput, path, artifact.DownloadURL)
	}

	resp.Output = updatedOutput
}

func (s *ServiceImpl) uploadSandboxImage(ctx context.Context, resp *Response, path string) (*plan.MediaArtifact, error) {
	content, err := s.readSandboxFile(ctx, resp, path)
	if err != nil {
		return nil, err
	}

	filename := filepath.Base(path)
	contentType := "image/png"
	if strings.EqualFold(filepath.Ext(filename), ".jpg") || strings.EqualFold(filepath.Ext(filename), ".jpeg") {
		contentType = "image/jpeg"
	}

	uploadReq := &media.UploadRequest{
		Content:        content,
		Filename:       filename,
		ContentType:    contentType,
		ResponseID:     resp.PublicID,
		UserID:         resp.UserID,
		ConversationID: safeString(resp.ConversationPublicID),
	}

	return s.mediaClient.UploadArtifact(ctx, uploadReq)
}

func (s *ServiceImpl) readSandboxFile(ctx context.Context, resp *Response, path string) ([]byte, error) {
	if s.mcpClient == nil {
		return nil, fmt.Errorf("mcp client not configured")
	}

	readFile := func(target string) (*tool.Result, error) {
		return s.mcpClient.CallTool(ctx, tool.CallRequest{
			Name: "aio_file_read",
			Arguments: map[string]interface{}{
				"path": target,
			},
			RequestID:      resp.PublicID,
			ConversationID: safeString(resp.ConversationPublicID),
			UserID:         resp.UserID,
		})
	}

	result, err := readFile(path)
	if err != nil || result == nil || result.IsError {
		fallback := fallbackImagePath(path)
		if fallback != "" && fallback != path {
			result, err = readFile(fallback)
			path = fallback
		}
	}

	if err != nil {
		return nil, err
	}
	if result == nil || result.IsError {
		return nil, fmt.Errorf("sandbox file read error: %s", result.Error)
	}

	rawText := firstTextContent(result.Content)
	if rawText == "" {
		return nil, fmt.Errorf("sandbox file read returned empty content")
	}

	return decodeSandboxContent(rawText, filepath.Ext(path))
}

func fallbackImagePath(path string) string {
	if strings.HasPrefix(path, "/tmp/") {
		return "/home/user/" + strings.TrimPrefix(path, "/tmp/")
	}
	return ""
}

func extractLocalImagePaths(output string) []string {
	if output == "" {
		return nil
	}

	re := regexp.MustCompile(`!\[[^\]]*]\(([^)]+)\)`)
	matches := re.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return nil
	}

	paths := make([]string, 0, len(matches))
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		path := normalizeLocalPath(match[1])
		if path == "" || seen[path] {
			continue
		}
		if strings.HasPrefix(path, "/tmp/") || strings.HasPrefix(path, "/home/user/") {
			paths = append(paths, path)
			seen[path] = true
		}
	}
	return paths
}

func normalizeLocalPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if strings.HasPrefix(trimmed, "file://") {
		return strings.TrimPrefix(trimmed, "file://")
	}
	return trimmed
}

func firstTextContent(contents []tool.MCPContent) string {
	for _, content := range contents {
		if content.Type == "text" && content.Text != "" {
			return content.Text
		}
	}
	return ""
}

func decodeSandboxContent(rawText string, ext string) ([]byte, error) {
	trimmed := strings.TrimSpace(rawText)
	if trimmed == "" {
		return nil, fmt.Errorf("empty sandbox content")
	}

	rawBytes := []byte(rawText)
	if isPNG(rawBytes) {
		return rawBytes, nil
	}

	if strings.EqualFold(ext, ".png") {
		if decoded, ok := tryDecodeBase64(trimmed); ok && isPNG(decoded) {
			return decoded, nil
		}
	}

	if decoded, ok := tryDecodeBase64(trimmed); ok {
		return decoded, nil
	}

	return rawBytes, nil
}

func tryDecodeBase64(value string) ([]byte, bool) {
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
		return decoded, true
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(value); err == nil {
		return decoded, true
	}
	return nil, false
}

func isPNG(data []byte) bool {
	return len(data) >= 8 &&
		data[0] == 0x89 &&
		data[1] == 0x50 &&
		data[2] == 0x4E &&
		data[3] == 0x47 &&
		data[4] == 0x0D &&
		data[5] == 0x0A &&
		data[6] == 0x1A &&
		data[7] == 0x0A
}

func safeString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// executePlanWithOrchestrator executes a plan using the tool orchestrator while updating plan progress.
func (s *ServiceImpl) executePlanWithOrchestrator(ctx context.Context, resp *Response, planResult *agent.PlanResult) error {
	if s.agentOrchestrator == nil {
		return errors.New("agent orchestrator not configured")
	}

	s.log.Info().
		Str("response_id", resp.PublicID).
		Str("plan_id", planResult.Plan.ID).
		Msg("starting plan execution with orchestrator")

	if err := s.agentOrchestrator.StartPlan(ctx, planResult.Plan.ID); err != nil {
		return fmt.Errorf("failed to start plan: %w", err)
	}

	maxIterations := planResult.Plan.EstimatedSteps * 2
	if maxIterations <= 0 {
		maxIterations = 50
	}

	var lastOutput interface{}

	for i := 0; i < maxIterations; i++ {
		result, err := s.agentOrchestrator.ExecuteNextStep(ctx, planResult.Plan.ID)
		if err != nil {
			now := time.Now()
			resp.Status = StatusFailed
			resp.Error = &ErrorDetails{Message: err.Error()}
			resp.CompletedAt = &now
			resp.UpdatedAt = now
			if updateErr := s.responses.Update(ctx, resp); updateErr != nil {
				s.log.Error().Err(updateErr).Msg("failed to update response after step failure")
			}
			return err
		}

		if result != nil && len(result.Output) > 0 {
			var decoded interface{}
			if unmarshalErr := json.Unmarshal(result.Output, &decoded); unmarshalErr == nil {
				lastOutput = decoded
			} else {
				lastOutput = string(result.Output)
			}
		}

		planStatus, statusErr := s.agentOrchestrator.GetStatus(ctx, planResult.Plan.ID)
		if statusErr != nil {
			s.log.Warn().Err(statusErr).Str("plan_id", planResult.Plan.ID).Msg("failed to get plan status")
			continue
		}

		switch planStatus.Status {
		case status.StatusCompleted:
			now := time.Now()
			resp.Status = StatusCompleted
			resp.CompletedAt = &now
			resp.UpdatedAt = now

			// Generate comprehensive output from plan results
			finalOutput := s.generatePlanOutput(ctx, planResult.Plan.ID)
			if finalOutput == "" {
				// Fallback to last step output if no aggregation
				if lastOutput != nil {
					finalOutput = lastOutput
				} else {
					finalOutput = "plan completed"
				}
			}
			resp.Output = finalOutput
			s.populateArtifactsAndCitations(ctx, resp, &planResult.Plan.ID)
			if err := s.responses.Update(ctx, resp); err != nil {
				return fmt.Errorf("failed to update response: %w", err)
			}
			s.log.Info().
				Str("response_id", resp.PublicID).
				Str("plan_id", planResult.Plan.ID).
				Msg("plan execution completed successfully")
			return nil
		case status.StatusFailed:
			now := time.Now()
			resp.Status = StatusFailed
			errMsg := "plan execution failed"
			if planStatus.LastError != nil {
				errMsg = *planStatus.LastError
			}
			resp.Error = &ErrorDetails{Message: errMsg}
			resp.CompletedAt = &now
			resp.UpdatedAt = now
			if err := s.responses.Update(ctx, resp); err != nil {
				return fmt.Errorf("failed to update response after failure: %w", err)
			}
			return fmt.Errorf("plan failed: %s", errMsg)
		case status.StatusWaitForUser:
			resp.Status = Status(status.StatusWaitForUser)
			resp.UpdatedAt = time.Now()
			if err := s.responses.Update(ctx, resp); err != nil {
				return fmt.Errorf("failed to update response while waiting for user: %w", err)
			}
			s.log.Info().
				Str("response_id", resp.PublicID).
				Str("plan_id", planResult.Plan.ID).
				Msg("plan paused waiting for user input")
			return nil
		}
	}

	return fmt.Errorf("plan execution exceeded maximum iterations (%d)", maxIterations)
}

// generatePlanOutput aggregates results from plan execution into a comprehensive output.
func (s *ServiceImpl) generatePlanOutput(ctx context.Context, planID string) interface{} {
	// Get plan with all tasks and steps
	planDetails, err := s.planService.GetPlanWithDetails(ctx, planID)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to get plan details for output generation")
		return ""
	}

	// Strategy 1: Look for final report generation step
	for _, task := range planDetails.Tasks {
		if task.TaskType == plan.TaskTypeGeneration || task.Title == "Report" {
			for _, step := range task.Steps {
				if step.Status == status.StatusCompleted && len(step.OutputData) > 0 {
					var stepOutput map[string]interface{}
					if err := json.Unmarshal(step.OutputData, &stepOutput); err == nil {
						if content, ok := stepOutput["content"].(string); ok && content != "" {
							s.log.Info().Msg("Using final report generation output")
							return content
						}
					}
				}
			}
		}
	}

	// Strategy 2: Aggregate results from all successful steps
	var results strings.Builder
	results.WriteString("# Research Analysis\n\n")

	for _, task := range planDetails.Tasks {
		hasOutput := false
		for _, step := range task.Steps {
			if step.Status == status.StatusCompleted && len(step.OutputData) > 0 {
				hasOutput = true
				break
			}
		}

		if !hasOutput {
			continue
		}

		results.WriteString(fmt.Sprintf("## %s\n\n", task.Title))

		for _, step := range task.Steps {
			if step.Status == status.StatusCompleted && len(step.OutputData) > 0 {
				var stepOutput map[string]interface{}
				if err := json.Unmarshal(step.OutputData, &stepOutput); err == nil {
					// Handle different output types
					if content, ok := stepOutput["content"].(string); ok {
						// LLM generated content
						results.WriteString(content)
						results.WriteString("\n\n")
					} else if resultsList, ok := stepOutput["results"].([]interface{}); ok {
						// Search results
						for i, r := range resultsList {
							if i >= 5 {
								break // Limit to top 5
							}
							if result, ok := r.(map[string]interface{}); ok {
								if title, ok := result["title"].(string); ok {
									results.WriteString(fmt.Sprintf("- **%s**", title))
									if snippet, ok := result["snippet"].(string); ok {
										results.WriteString(fmt.Sprintf(": %s", snippet))
									}
									if url, ok := result["url"].(string); ok {
										results.WriteString(fmt.Sprintf(" ([source](%s))", url))
									}
									results.WriteString("\n")
								}
							}
						}
						results.WriteString("\n")
					} else if _, ok := stepOutput["code"].(string); ok {
						// Code execution output
						results.WriteString("### Code Output\n\n```\n")
						if output, ok := stepOutput["output"].(string); ok {
							results.WriteString(output)
						}
						results.WriteString("\n```\n\n")
					}
				}
			}
		}
	}

	output := results.String()
	if output == "# Research Analysis\n\n" {
		// No content was added
		return ""
	}

	return output
}

func extractCodeBlock(content string) string {
	if content == "" {
		return ""
	}

	lower := strings.ToLower(content)
	startIdx := strings.Index(lower, "```python")
	if startIdx == -1 {
		startIdx = strings.Index(lower, "```")
		if startIdx == -1 {
			return ""
		}
		startIdx += len("```")
	} else {
		startIdx += len("```python")
	}

	endIdx := strings.Index(lower[startIdx:], "```")
	if endIdx == -1 {
		return ""
	}

	code := content[startIdx : startIdx+endIdx]
	return strings.TrimSpace(code)
}

func extractString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case fmt.Stringer:
		return v.String()
	default:
		return ""
	}
}

// extractUserMessage extracts the user message from the input.
func (s *ServiceImpl) extractUserMessage(input interface{}) string {
	if input == nil {
		return ""
	}
	switch v := input.(type) {
	case string:
		return v
	case []interface{}:
		// Handle array of message objects
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if role, _ := m["role"].(string); role == "user" {
					if content, ok := m["content"].(string); ok {
						return content
					}
				}
			}
		}
	}
	// Fallback: serialize to JSON
	if b, err := json.Marshal(input); err == nil {
		return string(b)
	}
	return ""
}

// generateLLMStepOutput creates step-specific output for LLM steps based on their purpose.
func (s *ServiceImpl) generateLLMStepOutput(task *plan.Task, step plan.Step, stepParams map[string]interface{}, result *tool.ExecuteResult, llmStepIndex int) map[string]interface{} {
	if result == nil || result.FinalMessage.Content == "" {
		return nil
	}

	action, _ := stepParams["action"].(string)
	description, _ := stepParams["description"].(string)
	finalContent := result.FinalMessage.GetContentAsString()

	output := map[string]interface{}{
		"type": "llm_response",
	}

	switch action {
	case "reasoning":
		// Synthesis/reasoning step - extract key themes or provide summary
		output["action"] = "reasoning"
		output["description"] = description
		output["content"] = generateReasoningSummary(finalContent, task.Title)

	case "generate_code":
		// Code generation step - extract just the code portion
		output["action"] = "generate_code"
		output["description"] = description
		code := extractCodeBlock(finalContent)
		if code != "" {
			output["content"] = code
			output["code_extracted"] = true
		} else {
			output["content"] = finalContent
			output["code_extracted"] = false
		}

	case "generate_content":
		// Report/content generation - this is typically the final output
		output["action"] = "generate_content"
		output["description"] = description
		output["content"] = finalContent

	default:
		// Generic LLM step
		output["action"] = action
		output["description"] = description
		// For the last LLM step, include full content; otherwise truncate
		if llmStepIndex == 0 || action == "" {
			output["content"] = finalContent
		} else {
			output["content"] = truncateContent(finalContent, 500)
		}
	}

	return output
}

// generateReasoningSummary creates a summary for reasoning/synthesis steps.
func generateReasoningSummary(content string, taskTitle string) string {
	// For synthesis steps, we want a brief summary, not the full content
	// Extract key points or first paragraph
	if len(content) <= 500 {
		return content
	}

	// Find the first paragraph or section
	lines := strings.Split(content, "\n\n")
	if len(lines) > 0 {
		firstParagraph := strings.TrimSpace(lines[0])
		if len(firstParagraph) > 100 {
			return firstParagraph + "..."
		}
	}

	return truncateContent(content, 500)
}

// truncateContent truncates content to a maximum length with ellipsis.
func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen-3] + "..."
}

// isStepOutputIndicatingFailure checks if step output indicates a failure.
func isStepOutputIndicatingFailure(output []byte) bool {
	if len(output) == 0 {
		return false
	}

	var outputMap map[string]interface{}
	if err := json.Unmarshal(output, &outputMap); err != nil {
		return false
	}

	// Check for is_error field
	if isError, ok := outputMap["is_error"].(bool); ok && isError {
		return true
	}

	// Check for status field indicating failure
	if statusStr, ok := outputMap["status"].(string); ok {
		if statusStr == "failed" || statusStr == "error" {
			return true
		}
	}

	// Check nested result for is_error
	if result, ok := outputMap["result"].(map[string]interface{}); ok {
		if isError, ok := result["is_error"].(bool); ok && isError {
			return true
		}

		// Check for error in content text (code execution results)
		if hasErrorInContent(result) {
			return true
		}
	}

	return false
}

// hasErrorInContent checks if result content contains error indicators.
func hasErrorInContent(result map[string]interface{}) bool {
	content, ok := result["content"].([]interface{})
	if !ok {
		return false
	}

	for _, c := range content {
		cMap, ok := c.(map[string]interface{})
		if !ok {
			continue
		}

		text, ok := cMap["text"].(string)
		if !ok || text == "" {
			continue
		}

		// Try to parse the text as JSON (code execution returns nested JSON)
		var nestedResult map[string]interface{}
		if err := json.Unmarshal([]byte(text), &nestedResult); err == nil {
			// Check for error_details in the nested result
			if errorDetails, ok := nestedResult["error_details"].(map[string]interface{}); ok {
				if errorName, ok := errorDetails["error_name"].(string); ok && errorName != "" {
					return true
				}
			}

			// Check for success=false in nested result
			if success, ok := nestedResult["success"].(bool); ok && !success {
				return true
			}

			// Check for error field in nested result
			if errStr, ok := nestedResult["error"].(string); ok && errStr != "" {
				return true
			}
		}

		// Check for Python error patterns in raw text
		if containsCodeExecutionError(text) {
			return true
		}
	}

	return false
}

// containsCodeExecutionError checks for common Python/code error patterns.
func containsCodeExecutionError(text string) bool {
	errorPatterns := []string{
		"ModuleNotFoundError",
		"ImportError",
		"SyntaxError",
		"NameError",
		"TypeError",
		"ValueError",
		"AttributeError",
		"KeyError",
		"IndexError",
		"ZeroDivisionError",
		"FileNotFoundError",
		"PermissionError",
		"RuntimeError",
		"RecursionError",
		"MemoryError",
		"Traceback (most recent call last)",
	}

	for _, pattern := range errorPatterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}

	return false
}

// extractErrorFromStepOutput extracts the error message from step output.
func extractErrorFromStepOutput(output []byte) string {
	if len(output) == 0 {
		return "unknown error"
	}

	var outputMap map[string]interface{}
	if err := json.Unmarshal(output, &outputMap); err != nil {
		return "failed to parse output"
	}

	// Check for error field
	if errStr, ok := outputMap["error"].(string); ok && errStr != "" {
		return errStr
	}

	// Check nested result for content with error text
	if result, ok := outputMap["result"].(map[string]interface{}); ok {
		if content, ok := result["content"].([]interface{}); ok {
			for _, c := range content {
				if cMap, ok := c.(map[string]interface{}); ok {
					if text, ok := cMap["text"].(string); ok && text != "" {
						// Truncate if too long
						if len(text) > 500 {
							return text[:500] + "..."
						}
						return text
					}
				}
			}
		}
	}

	return "execution failed"
}

// createStepDetails creates StepDetail records to link a step to conversation items.
// This enables traceability from plan steps to the actual conversation data.
func (s *ServiceImpl) createStepDetails(ctx context.Context, step plan.Step, callID string, execID uint, tracker *conversationItemTracker, assistantMsgIndex int) {
	if tracker == nil {
		return
	}

	switch step.Action {
	case plan.ActionTypeToolCall:
		// For tool_call steps, create details for:
		// 1. The tool result message (DetailTypeToolResult)
		if callID != "" {
			if item := tracker.getItemByToolCallID(callID); item != nil && item.ID != 0 {
				itemIDStr := fmt.Sprintf("%d", item.ID)
				callIDPtr := &callID
				execIDStr := fmt.Sprintf("%d", execID)
				var execIDPtr *string
				if execID != 0 {
					execIDPtr = &execIDStr
				}
				detail := &plan.StepDetail{
					ID:                 generateStepDetailID(),
					DetailType:         plan.DetailTypeToolResult,
					ConversationItemID: &itemIDStr,
					ToolCallID:         callIDPtr,
					ToolExecutionID:    execIDPtr,
				}
				if err := s.planService.AddStepDetail(ctx, step.ID, detail); err != nil {
					s.log.Warn().Err(err).Str("step_id", step.ID).Msg("failed to add tool result step detail")
				}
			}
		}

	case plan.ActionTypeLLMCall:
		// For llm_call steps, create detail for the assistant message
		assistantItems := tracker.getItemsByRole("assistant")
		if assistantMsgIndex < len(assistantItems) {
			item := assistantItems[assistantMsgIndex]
			if item != nil && item.ID != 0 {
				itemIDStr := fmt.Sprintf("%d", item.ID)
				detail := &plan.StepDetail{
					ID:                 generateStepDetailID(),
					DetailType:         plan.DetailTypeMessage,
					ConversationItemID: &itemIDStr,
				}
				if err := s.planService.AddStepDetail(ctx, step.ID, detail); err != nil {
					s.log.Warn().Err(err).Str("step_id", step.ID).Msg("failed to add llm message step detail")
				}
			}
		}
	}
}

// generateStepDetailID generates a unique ID for step details.
func generateStepDetailID() string {
	return fmt.Sprintf("sd_%d", time.Now().UnixNano())
}
