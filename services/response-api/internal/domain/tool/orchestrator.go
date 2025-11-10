package tool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"jan-server/services/response-api/internal/domain/llm"
)

var (
	// ErrToolDepthExceeded is returned when the orchestrator hits the max recursion depth.
	ErrToolDepthExceeded = errors.New("tool orchestration depth exceeded")
)

// Orchestrator coordinates LLM reasoning with MCP tool execution until a final answer is produced.
type Orchestrator struct {
	llmProvider     llm.Provider
	mcpClient       MCPClient
	maxDepth        int
	toolCallTimeout time.Duration
}

// NewOrchestrator constructs a tool orchestrator instance.
func NewOrchestrator(llmProvider llm.Provider, mcpClient MCPClient, maxDepth int, toolCallTimeout time.Duration) *Orchestrator {
	return &Orchestrator{
		llmProvider:     llmProvider,
		mcpClient:       mcpClient,
		maxDepth:        maxDepth,
		toolCallTimeout: toolCallTimeout,
	}
}

// ExecuteParams contains the data needed to start the orchestration loop.
type ExecuteParams struct {
	Ctx             context.Context
	Model           string
	Messages        []llm.ChatMessage
	Temperature     *float64
	MaxTokens       *int
	ToolChoice      *llm.ToolChoice
	ToolDefinitions []llm.ToolDefinition
	StreamObserver  StreamObserver
}

// ExecuteResult captures the final assistant message and tool execution records.
type ExecuteResult struct {
	FinalMessage llm.ChatMessage
	Messages     []llm.ChatMessage
	Usage        *llm.Usage
	Executions   []Execution
}

// Execute drains the orchestration loop until the assistant responds without requesting tools.
func (o *Orchestrator) Execute(params ExecuteParams) (*ExecuteResult, error) {
	messages := append([]llm.ChatMessage(nil), params.Messages...)
	var executions []Execution

	for depth := 0; depth < o.maxDepth; depth++ {
		req := llm.ChatCompletionRequest{
			Model:       params.Model,
			Messages:    messages,
			Tools:       params.ToolDefinitions,
			ToolChoice:  params.ToolChoice,
			Temperature: params.Temperature,
			MaxTokens:   params.MaxTokens,
			Stream:      false,
		}
		req.Stream = params.StreamObserver != nil

		var choice llm.ChatCompletionChoice
		var usage *llm.Usage

		if params.StreamObserver != nil {
			streamChoice, err := o.streamChatCompletion(params.Ctx, req, params.StreamObserver)
			if err != nil {
				return nil, err
			}
			choice = *streamChoice
		} else {
			resp, err := o.llmProvider.CreateChatCompletion(params.Ctx, req)
			if err != nil {
				return nil, err
			}
			if len(resp.Choices) == 0 {
				return nil, errors.New("llm returned no choices")
			}
			choice = resp.Choices[0]
			usage = resp.Usage
		}

		messages = append(messages, choice.Message)

		if len(choice.Message.ToolCalls) == 0 {
			return &ExecuteResult{
				FinalMessage: choice.Message,
				Messages:     messages,
				Usage:        usage,
				Executions:   executions,
			}, nil
		}

		for _, call := range choice.Message.ToolCalls {
			parsedCall, err := ParseToolCall(call)
			if err != nil {
				return nil, fmt.Errorf("parse tool call: %w", err)
			}

			execution := Execution{
				CallID:         parsedCall.ID,
				ToolName:       parsedCall.Name,
				Arguments:      parsedCall.Arguments,
				Status:         ExecutionStatusRunning,
				ExecutionOrder: len(executions) + 1,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}

			if params.StreamObserver != nil {
				params.StreamObserver.OnToolCall(parsedCall)
			}

			callCtx := params.Ctx
			var cancel context.CancelFunc
			if o.toolCallTimeout > 0 {
				callCtx, cancel = context.WithTimeout(callCtx, o.toolCallTimeout)
			}

			result, err := o.mcpClient.CallTool(callCtx, parsedCall.Name, parsedCall.Arguments)
			if cancel != nil {
				cancel()
			}
			if err != nil {
				execution.Status = ExecutionStatusFailed
				execution.ErrorMessage = err.Error()
			} else {
				execution.Status = ExecutionStatusCompleted
				execution.Result = result
				if result != nil && result.IsError {
					execution.Status = ExecutionStatusFailed
					execution.ErrorMessage = result.Error
				}
			}
			execution.UpdatedAt = time.Now()
			executions = append(executions, execution)

			if params.StreamObserver != nil {
				params.StreamObserver.OnToolResult(parsedCall.ID, execution.Result)
			}

			messages = append(messages, toolResultToMessage(parsedCall.ID, execution.Result, execution.ErrorMessage))
		}
	}

	return nil, ErrToolDepthExceeded
}

func (o *Orchestrator) streamChatCompletion(ctx context.Context, req llm.ChatCompletionRequest, observer StreamObserver) (*llm.ChatCompletionChoice, error) {
	stream, err := o.llmProvider.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	accumulator := newStreamAccumulator()

	for {
		delta, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if observer != nil && delta != nil {
			observer.OnDelta(*delta)
		}
		accumulator.Apply(delta)
	}

	choice := accumulator.Result()
	if choice == nil {
		return nil, errors.New("stream produced no choices")
	}
	return choice, nil
}

func toolResultToMessage(toolCallID string, result *Result, errorMessage string) llm.ChatMessage {
	content := buildContentFromResult(result, errorMessage)
	return llm.ChatMessage{
		Role:       "tool",
		Content:    content,
		ToolCallID: &toolCallID,
	}
}

func buildContentFromResult(result *Result, errorMessage string) interface{} {
	if result == nil {
		return map[string]string{
			"type": "text",
			"text": errorMessage,
		}
	}

	if result.IsError {
		return map[string]string{
			"type": "text",
			"text": firstNonEmpty(errorMessage, "tool execution returned an error"),
		}
	}

	var sb strings.Builder
	for _, c := range result.Content {
		if c.Type == "text" {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(c.Text)
		}
	}

	text := sb.String()
	if text == "" {
		text = "[tool execution completed]"
	}

	return map[string]string{
		"type": "text",
		"text": text,
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

type streamAccumulator struct {
	choices map[int]*choiceAccumulator
}

func newStreamAccumulator() *streamAccumulator {
	return &streamAccumulator{
		choices: make(map[int]*choiceAccumulator),
	}
}

func (a *streamAccumulator) Apply(delta *llm.ChatCompletionDelta) {
	if delta == nil {
		return
	}
	for _, choice := range delta.Choices {
		acc := a.ensure(choice.Index)
		acc.apply(choice)
	}
}

func (a *streamAccumulator) ensure(index int) *choiceAccumulator {
	if acc, ok := a.choices[index]; ok {
		return acc
	}
	acc := &choiceAccumulator{
		role:      "assistant",
		toolCalls: make(map[string]*toolCallAccumulator),
	}
	a.choices[index] = acc
	return acc
}

func (a *streamAccumulator) Result() *llm.ChatCompletionChoice {
	if len(a.choices) == 0 {
		return nil
	}
	acc := a.choices[0]
	choice := acc.build(0)
	return &choice
}

type choiceAccumulator struct {
	role         string
	finishReason string
	content      strings.Builder
	toolCalls    map[string]*toolCallAccumulator
	toolOrder    []string
}

func (c *choiceAccumulator) apply(choice llm.ChatCompletionDeltaChoice) {
	if choice.Delta.Role != "" {
		c.role = choice.Delta.Role
	}

	if choice.Delta.Content != nil {
		c.appendContent(choice.Delta.Content)
	}

	if len(choice.Delta.ToolCalls) > 0 {
		for idx, call := range choice.Delta.ToolCalls {
			c.addOrUpdateToolCall(idx, call)
		}
	}

	if choice.FinishReason != "" {
		c.finishReason = choice.FinishReason
	}
}

func (c *choiceAccumulator) appendContent(content interface{}) {
	switch v := content.(type) {
	case string:
		c.content.WriteString(v)
	case []interface{}:
		for _, item := range v {
			c.appendContent(item)
		}
	case map[string]interface{}:
		if text, ok := v["text"].(string); ok {
			c.content.WriteString(text)
		}
	case nil:
		return
	default:
		c.content.WriteString(fmt.Sprint(v))
	}
}

func (c *choiceAccumulator) addOrUpdateToolCall(idx int, call llm.ToolCall) {
	id := call.ID
	if id == "" {
		id = fmt.Sprintf("tool_%d", len(c.toolOrder)+idx)
	}

	builder, ok := c.toolCalls[id]
	if !ok {
		builder = &toolCallAccumulator{}
		builder.call.ID = id
		c.toolCalls[id] = builder
		c.toolOrder = append(c.toolOrder, id)
	}

	if call.Type != "" {
		builder.call.Type = call.Type
	}
	if call.Function.Name != "" {
		builder.call.Function.Name = call.Function.Name
	}
	if len(call.Function.Arguments) > 0 {
		builder.args.Write(call.Function.Arguments)
		builder.call.Function.Arguments = json.RawMessage(builder.args.String())
	}
}

func (c *choiceAccumulator) build(index int) llm.ChatCompletionChoice {
	message := llm.ChatMessage{
		Role: c.role,
	}
	if c.content.Len() > 0 {
		message.Content = c.content.String()
	}
	if len(c.toolOrder) > 0 {
		message.ToolCalls = make([]llm.ToolCall, 0, len(c.toolOrder))
		for _, id := range c.toolOrder {
			builder := c.toolCalls[id]
			call := builder.call
			if len(call.Function.Arguments) == 0 && builder.args.Len() > 0 {
				call.Function.Arguments = json.RawMessage(builder.args.String())
			}
			message.ToolCalls = append(message.ToolCalls, call)
		}
	}

	return llm.ChatCompletionChoice{
		Index:        index,
		Message:      message,
		FinishReason: c.finishReason,
	}
}

type toolCallAccumulator struct {
	call llm.ToolCall
	args strings.Builder
}
