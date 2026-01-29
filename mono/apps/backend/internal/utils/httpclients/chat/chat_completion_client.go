package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"jan-server/mono/apps/backend/internal/utils/platformerrors"

	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"resty.dev/v3"
)

const (
	defaultStreamTimeout = 600 * time.Second // Default to 10 minutes for long requests
	channelBufferSize    = 100
	errorBufferSize      = 10
	dataPrefix           = "data: "
	doneMarker           = "[DONE]"
	newlineChar          = "\n"
	scannerInitialBuffer = 12 * 1024        // 12KB
	scannerMaxBuffer     = 10 * 1024 * 1024 // 10MB
)

type StreamOption func(*resty.Request)

// BeforeDoneCallback is called before writing [DONE] marker
type BeforeDoneCallback func(*gin.Context) error

type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChoiceDelta struct {
	Content          string               `json:"content"`
	ReasoningContent string               `json:"reasoning_content"`
	FunctionCall     *openai.FunctionCall `json:"function_call,omitempty"`
	ToolCalls        []openai.ToolCall    `json:"tool_calls,omitempty"`
}

type StreamChoice struct {
	Delta ChoiceDelta `json:"delta"`
}

func WithHeader(key, value string) StreamOption {
	return func(r *resty.Request) {
		if strings.TrimSpace(key) == "" {
			return
		}
		if value == "" {
			r.SetHeader(key, "")
			return
		}
		r.SetHeader(key, value)
	}
}

func WithAcceptEncodingIdentity() StreamOption {
	return WithHeader("Accept-Encoding", "identity")
}

type ChatCompletionClient struct {
	client        *resty.Client
	baseURL       string
	name          string
	streamTimeout time.Duration
}

// CompletionRequest extends the OpenAI chat request with provider-specific fields.
type CompletionRequest struct {
	openai.ChatCompletionRequest
	TopK              *int     `json:"top_k,omitempty"`
	RepetitionPenalty *float32 `json:"repetition_penalty,omitempty"`
}

type functionCallAccumulator struct {
	Name      string
	Arguments string
	Complete  bool
}

type toolCallAccumulator struct {
	ID       string
	Type     string
	Index    int
	Function struct {
		Name      string
		Arguments string
	}
	Complete bool
}

// ClientOption is a functional option for configuring ChatCompletionClient
type ClientOption func(*ChatCompletionClient)

// WithStreamTimeout sets a custom stream timeout
func WithStreamTimeout(timeout time.Duration) ClientOption {
	return func(c *ChatCompletionClient) {
		if timeout > 0 {
			c.streamTimeout = timeout
		}
	}
}

func NewChatCompletionClient(client *resty.Client, name, baseURL string, opts ...ClientOption) *ChatCompletionClient {
	c := &ChatCompletionClient{
		client:        client,
		baseURL:       normalizeBaseURL(baseURL),
		name:          name,
		streamTimeout: defaultStreamTimeout,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *ChatCompletionClient) CreateChatCompletion(ctx context.Context, apiKey string, request CompletionRequest) (*openai.ChatCompletionResponse, error) {
	// Sanitize messages to remove invalid parts that cause provider validation errors
	request.Messages = SanitizeMessages(request.Messages)

	// Clear ToolChoice if there are no tools - providers reject tool_choice without tools
	if len(request.Tools) == 0 {
		request.ToolChoice = nil
	}

	// Start OpenTelemetry span for tracking
	ctx, span := otel.Tracer("chat-completion-client").Start(ctx, "CreateChatCompletion",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("llm.provider", c.name),
			attribute.String("llm.model", request.Model),
			attribute.Int("llm.message_count", len(request.Messages)),
			attribute.Bool("llm.stream", false),
		),
	)
	defer span.End()

	// Add optional parameters as attributes
	if request.Temperature != 0 {
		span.SetAttributes(attribute.Float64("llm.temperature", float64(request.Temperature)))
	}
	if request.MaxTokens != 0 {
		span.SetAttributes(attribute.Int("llm.max_tokens", request.MaxTokens))
	}
	if request.TopP != 0 {
		span.SetAttributes(attribute.Float64("llm.top_p", float64(request.TopP)))
	}
	if request.PresencePenalty != 0 {
		span.SetAttributes(attribute.Float64("llm.presence_penalty", float64(request.PresencePenalty)))
	}
	if request.FrequencyPenalty != 0 {
		span.SetAttributes(attribute.Float64("llm.frequency_penalty", float64(request.FrequencyPenalty)))
	}

	start := time.Now()

	var respBody openai.ChatCompletionResponse
	resp, err := c.prepareRequest(ctx, apiKey).
		SetBody(request).
		SetResult(&respBody).
		Post(c.endpoint("/chat/completions"))

	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.Int64("llm.duration_ms", duration.Milliseconds()))
		return nil, err
	}
	if resp.IsError() {
		reqErr := c.errorFromResponse(ctx, resp, "request failed")
		span.RecordError(reqErr)
		span.SetStatus(codes.Error, reqErr.Error())
		span.SetAttributes(
			attribute.Int("http.status_code", resp.StatusCode()),
			attribute.Int64("llm.duration_ms", duration.Milliseconds()),
		)
		return nil, reqErr
	}

	// Record token usage and timing in span
	span.SetAttributes(
		attribute.Int("llm.usage.prompt_tokens", respBody.Usage.PromptTokens),
		attribute.Int("llm.usage.completion_tokens", respBody.Usage.CompletionTokens),
		attribute.Int("llm.usage.total_tokens", respBody.Usage.TotalTokens),
		attribute.Int64("llm.duration_ms", duration.Milliseconds()),
		attribute.Int("http.status_code", resp.StatusCode()),
	)

	// Add finish reason if available
	if len(respBody.Choices) > 0 {
		span.SetAttributes(attribute.String("llm.finish_reason", string(respBody.Choices[0].FinishReason)))
	}

	// Add reasoning tokens if available
	if respBody.Usage.CompletionTokensDetails != nil && respBody.Usage.CompletionTokensDetails.ReasoningTokens > 0 {
		span.SetAttributes(attribute.Int("llm.usage.reasoning_tokens", respBody.Usage.CompletionTokensDetails.ReasoningTokens))
	}

	span.SetStatus(codes.Ok, "completion successful")
	span.AddEvent("chat_completion_completed", trace.WithAttributes(
		attribute.Int("response.choice_count", len(respBody.Choices)),
	))

	return &respBody, nil
}

func (c *ChatCompletionClient) CreateChatCompletionStream(ctx context.Context, apiKey string, request CompletionRequest, opts ...StreamOption) (io.ReadCloser, error) {
	resp, err := c.doStreamingRequest(ctx, apiKey, request, opts...)
	if err != nil {
		return nil, err
	}

	reader, writer := io.Pipe()

	go func() {
		defer func() {
			_ = resp.RawResponse.Body.Close()
		}()

		if _, copyErr := io.Copy(writer, resp.RawResponse.Body); copyErr != nil {
			_ = writer.CloseWithError(copyErr)
			return
		}
		_ = writer.Close()
	}()

	return reader, nil
}

func (c *ChatCompletionClient) StreamChatCompletionToContext(reqCtx *gin.Context, apiKey string, request openai.ChatCompletionRequest, opts ...StreamOption) (*openai.ChatCompletionResponse, error) {
	return c.StreamChatCompletionToContextWithCallback(reqCtx, apiKey, CompletionRequest{ChatCompletionRequest: request}, nil, opts...)
}

func (c *ChatCompletionClient) StreamChatCompletionToContextWithCallback(reqCtx *gin.Context, apiKey string, request CompletionRequest, beforeDone BeforeDoneCallback, opts ...StreamOption) (*openai.ChatCompletionResponse, error) {
	// Start OpenTelemetry span for tracking streaming completion
	ctx := reqCtx.Request.Context()
	ctx, span := otel.Tracer("chat-completion-client").Start(ctx, "StreamChatCompletion",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("llm.provider", c.name),
			attribute.String("llm.model", request.Model),
			attribute.Int("llm.message_count", len(request.Messages)),
			attribute.Bool("llm.stream", true),
		),
	)
	defer span.End()

	// Add optional parameters as attributes
	if request.Temperature != 0 {
		span.SetAttributes(attribute.Float64("llm.temperature", float64(request.Temperature)))
	}
	if request.MaxTokens != 0 {
		span.SetAttributes(attribute.Int("llm.max_tokens", request.MaxTokens))
	}
	if request.TopP != 0 {
		span.SetAttributes(attribute.Float64("llm.top_p", float64(request.TopP)))
	}
	if request.PresencePenalty != 0 {
		span.SetAttributes(attribute.Float64("llm.presence_penalty", float64(request.PresencePenalty)))
	}
	if request.FrequencyPenalty != 0 {
		span.SetAttributes(attribute.Float64("llm.frequency_penalty", float64(request.FrequencyPenalty)))
	}

	start := time.Now()

	// force to true to collect tokens
	request.StreamOptions = &openai.StreamOptions{
		IncludeUsage: true,
	}

	streamCtx, cancel := context.WithTimeout(ctx, c.streamTimeout)
	defer cancel()

	c.SetupSSEHeaders(reqCtx)

	dataChan := make(chan string, channelBufferSize)
	errChan := make(chan error, errorBufferSize)

	var wg sync.WaitGroup
	wg.Add(1)

	go c.streamResponseToChannel(streamCtx, apiKey, request, dataChan, errChan, &wg, opts)

	var contentBuilder strings.Builder
	var reasoningBuilder strings.Builder
	functionCallAccumulator := make(map[int]*functionCallAccumulator)
	toolCallAccumulator := make(map[int]*toolCallAccumulator)

	// Track streaming metrics
	var chunksReceived int
	var totalUsage *TokenUsage

	streamingComplete := false

	for !streamingComplete {
		select {
		case line, ok := <-dataChan:
			if !ok {
				streamingComplete = true
				break
			}

			chunksReceived++

			// Check if this is the [DONE] marker BEFORE writing it
			// Check if this is the [DONE] marker BEFORE writing it
			if data, found := strings.CutPrefix(line, dataPrefix); found {
				if data == doneMarker {
					// Call the beforeDone callback BEFORE sending [DONE]
					if beforeDone != nil {
						_ = beforeDone(reqCtx)
					}
					// Now write the [DONE] marker
					if err := c.writeSSELine(reqCtx, line); err != nil {
						cancel()
						wg.Wait()
						span.RecordError(err)
						span.SetStatus(codes.Error, "failed to write SSE done marker")
						return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "unable to write SSE line")
					}
					streamingComplete = true
					cancel()
					break
				}
			}

			// Write the line for non-[DONE] events
			if err := c.writeSSELine(reqCtx, line); err != nil {
				cancel()
				wg.Wait()
				span.RecordError(err)
				span.SetStatus(codes.Error, "failed to write SSE line")
				return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "unable to write SSE line")
			}

			// Process the data chunk
			if data, found := strings.CutPrefix(line, dataPrefix); found {
				chunksReceived++

				choice, usage := c.processStreamChunkForChannel(data)

				// Capture final usage if available
				if usage != nil {
					totalUsage = usage
				}

				if choice != nil {
					if choice.Delta.Content != "" {
						contentBuilder.WriteString(choice.Delta.Content)
					}

					if choice.Delta.ReasoningContent != "" {
						reasoningBuilder.WriteString(choice.Delta.ReasoningContent)
					}

					if choice.Delta.FunctionCall != nil {
						c.handleStreamingFunctionCall(choice.Delta.FunctionCall, functionCallAccumulator)
					}

					if len(choice.Delta.ToolCalls) > 0 {
						c.handleStreamingToolCall(&choice.Delta.ToolCalls[0], toolCallAccumulator)
					}
				}
			}

		case err, ok := <-errChan:
			if ok && err != nil {
				cancel()
				wg.Wait()
				span.RecordError(err)
				span.SetStatus(codes.Error, "streaming error")
				return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, err, "streaming error")
			}

		case <-streamCtx.Done():
			wg.Wait()
			span.RecordError(streamCtx.Err())
			span.SetStatus(codes.Error, "streaming context cancelled")
			return nil, platformerrors.AsError(ctx, platformerrors.LayerDomain, streamCtx.Err(), "streaming context cancelled")

		case <-reqCtx.Request.Context().Done():
			cancel()
			wg.Wait()
			span.RecordError(reqCtx.Request.Context().Err())
			span.SetStatus(codes.Error, "client request cancelled")
			return nil, platformerrors.AsError(reqCtx.Request.Context(), platformerrors.LayerDomain, reqCtx.Request.Context().Err(), "client request cancelled")
		}
	}

	cancel()
	wg.Wait()

	close(dataChan)
	close(errChan)

	duration := time.Since(start)

	response := c.buildCompleteResponse(
		contentBuilder.String(),
		reasoningBuilder.String(),
		functionCallAccumulator,
		toolCallAccumulator,
		request.Model,
		request,
	)

	// Record streaming metrics in span
	span.SetAttributes(
		attribute.Int("llm.streaming.chunks_received", chunksReceived),
		attribute.Int64("llm.duration_ms", duration.Milliseconds()),
	)

	// Add token usage if available from streaming
	if totalUsage != nil {
		span.SetAttributes(
			attribute.Int("llm.usage.prompt_tokens", totalUsage.PromptTokens),
			attribute.Int("llm.usage.completion_tokens", totalUsage.CompletionTokens),
			attribute.Int("llm.usage.total_tokens", totalUsage.TotalTokens),
		)
	} else {
		// Use estimated usage from response
		span.SetAttributes(
			attribute.Int("llm.usage.prompt_tokens", response.Usage.PromptTokens),
			attribute.Int("llm.usage.completion_tokens", response.Usage.CompletionTokens),
			attribute.Int("llm.usage.total_tokens", response.Usage.TotalTokens),
		)
	}

	// Add finish reason if available
	if len(response.Choices) > 0 {
		span.SetAttributes(attribute.String("llm.finish_reason", string(response.Choices[0].FinishReason)))
	}

	span.SetStatus(codes.Ok, "streaming completion successful")
	span.AddEvent("streaming_completed", trace.WithAttributes(
		attribute.Int("chunks.total", chunksReceived),
		attribute.Int("content.length", len(contentBuilder.String())),
	))

	return &response, nil
}

func (c *ChatCompletionClient) SetupSSEHeaders(reqCtx *gin.Context) {
	if reqCtx == nil {
		return
	}

	reqCtx.Header("Content-Type", "text/event-stream")
	reqCtx.Header("Cache-Control", "no-cache")
	reqCtx.Header("Connection", "keep-alive")
	reqCtx.Header("Access-Control-Allow-Origin", "*")
	reqCtx.Header("Access-Control-Allow-Headers", "Cache-Control")
	reqCtx.Header("Transfer-Encoding", "chunked")
	reqCtx.Writer.WriteHeaderNow()
}

func (c *ChatCompletionClient) prepareRequest(ctx context.Context, apiKey string) *resty.Request {
	req := c.client.R().SetContext(ctx)
	req.SetHeader("Content-Type", "application/json")
	if strings.TrimSpace(apiKey) != "" {
		req.SetHeader("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	}
	return req
}

func (c *ChatCompletionClient) endpoint(path string) string {
	if path == "" {
		return c.baseURL
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if c.baseURL == "" {
		return path
	}
	if strings.HasPrefix(path, "/") {
		return c.baseURL + path
	}
	return c.baseURL + "/" + path
}

func (c *ChatCompletionClient) errorFromResponse(ctx context.Context, resp *resty.Response, message string) error {
	if resp == nil || resp.RawResponse == nil || resp.RawResponse.Body == nil {
		return platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeExternal, message, nil, "3476dd55-5fc0-4653-bd10-665895ecc099")
	}
	defer resp.RawResponse.Body.Close()
	body, err := io.ReadAll(resp.RawResponse.Body)
	if err != nil {
		return platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeExternal, message, nil, "8cd2cae7-9ad9-40fe-ac00-8f9b24251064")
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeExternal, message, nil, "b8797de4-38cb-4bd9-9ae8-b9a04e70f6ab")
	}
	return platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeExternal, fmt.Sprintf("%s: %s", message, trimmed), nil, "a1f46e0d-4017-4411-ac05-987946c3066d")
}

func (c *ChatCompletionClient) doStreamingRequest(ctx context.Context, apiKey string, request CompletionRequest, opts ...StreamOption) (*resty.Response, error) {
	// Sanitize messages to remove invalid parts that cause provider validation errors
	request.Messages = SanitizeMessages(request.Messages)

	// Clear ToolChoice if there are no tools - providers reject tool_choice without tools
	if len(request.Tools) == 0 {
		request.ToolChoice = nil
	}

	req := c.prepareRequest(ctx, apiKey).
		SetBody(request).
		SetDoNotParseResponse(true)

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(req)
	}

	if req.Header.Get("Accept-Encoding") == "" {
		req.SetHeader("Accept-Encoding", "identity")
	}

	resp, err := req.Post(c.endpoint("/chat/completions"))
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, c.errorFromResponse(ctx, resp, "streaming request failed")
	}
	if resp.RawResponse == nil || resp.RawResponse.Body == nil {
		return nil, platformerrors.NewError(ctx, platformerrors.LayerDomain, platformerrors.ErrorTypeExternal, "streaming request failed: empty response body", nil, "1b3ab461-dbf9-4034-8abb-dfc6ea8486c5")
	}

	return resp, nil
}

func (c *ChatCompletionClient) streamResponseToChannel(ctx context.Context, apiKey string, request CompletionRequest, dataChan chan<- string, errChan chan<- error, wg *sync.WaitGroup, opts []StreamOption) {
	defer wg.Done()

	resp, err := c.doStreamingRequest(ctx, apiKey, request, opts...)
	if err != nil {
		c.sendAsyncError(errChan, err)
		return
	}

	defer func() {
		_ = resp.RawResponse.Body.Close()
	}()

	scanner := bufio.NewScanner(resp.RawResponse.Body)
	scanner.Buffer(make([]byte, 0, scannerInitialBuffer), scannerMaxBuffer)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			c.sendAsyncError(errChan, ctx.Err())
			return
		default:
		}

		line := scanner.Text()

		select {
		case dataChan <- line:
		case <-ctx.Done():
			c.sendAsyncError(errChan, ctx.Err())
			return
		}
	}

	if err := scanner.Err(); err != nil {
		c.sendAsyncError(errChan, err)
	}
}

func (c *ChatCompletionClient) writeSSELine(reqCtx *gin.Context, line string) error {
	if reqCtx == nil {
		return platformerrors.NewError(context.Background(), platformerrors.LayerDomain, platformerrors.ErrorTypeValidation, "nil gin context provided", nil, "8ee6e88f-07e9-49e5-9c7a-6e1dfe151456")
	}
	_, err := reqCtx.Writer.Write([]byte(line + newlineChar))
	if err != nil {
		return err
	}
	reqCtx.Writer.Flush()
	return nil
}

func (c *ChatCompletionClient) processStreamChunkForChannel(data string) (*StreamChoice, *TokenUsage) {
	var streamData struct {
		Choices []StreamChoice `json:"choices"`
		Usage   *TokenUsage    `json:"usage"`
	}

	if err := json.Unmarshal([]byte(data), &streamData); err != nil {
		return nil, nil
	}

	result := &StreamChoice{
		Delta: ChoiceDelta{},
	}

	for _, choice := range streamData.Choices {
		if choice.Delta.Content != "" {
			result.Delta.Content += choice.Delta.Content
		}

		if choice.Delta.ReasoningContent != "" {
			result.Delta.ReasoningContent += choice.Delta.ReasoningContent
		}

		if choice.Delta.FunctionCall != nil {
			result.Delta.FunctionCall = choice.Delta.FunctionCall
		}

		if len(choice.Delta.ToolCalls) > 0 {
			// TODO: Handle multiple tool calls if needed
			result.Delta.ToolCalls = choice.Delta.ToolCalls
		}
	}

	return result, streamData.Usage
}

func (c *ChatCompletionClient) handleStreamingFunctionCall(functionCall *openai.FunctionCall, accumulator map[int]*functionCallAccumulator) {
	if functionCall == nil {
		return
	}

	index := 0
	if accumulator[index] == nil {
		accumulator[index] = &functionCallAccumulator{}
	}

	if functionCall.Name != "" {
		accumulator[index].Name = functionCall.Name
	}
	if functionCall.Arguments != "" {
		accumulator[index].Arguments += functionCall.Arguments
	}

	if accumulator[index].Name != "" && accumulator[index].Arguments != "" && strings.HasSuffix(accumulator[index].Arguments, "}") {
		accumulator[index].Complete = true
	}
}

func (c *ChatCompletionClient) handleStreamingToolCall(toolCall *openai.ToolCall, accumulator map[int]*toolCallAccumulator) {
	if toolCall == nil || toolCall.Index == nil {
		return
	}

	index := *toolCall.Index
	if accumulator[index] == nil {
		accumulator[index] = &toolCallAccumulator{
			ID:    toolCall.ID,
			Type:  string(toolCall.Type),
			Index: index,
		}
	}

	if toolCall.Function.Name != "" {
		accumulator[index].Function.Name = toolCall.Function.Name
	}
	if toolCall.Function.Arguments != "" {
		accumulator[index].Function.Arguments += toolCall.Function.Arguments
	}

	if accumulator[index].Function.Name != "" && accumulator[index].Function.Arguments != "" && strings.HasSuffix(accumulator[index].Function.Arguments, "}") {
		accumulator[index].Complete = true
	}
}

func (c *ChatCompletionClient) buildCompleteResponse(content string, reasoning string, functionCallAccumulator map[int]*functionCallAccumulator, toolCallAccumulator map[int]*toolCallAccumulator, model string, request CompletionRequest) openai.ChatCompletionResponse {
	message := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: content,
	}

	if reasoning != "" {
		message.ReasoningContent = reasoning
	}

	finishReason := openai.FinishReasonStop

	if len(functionCallAccumulator) > 0 {
		for _, acc := range functionCallAccumulator {
			if acc != nil && acc.Complete {
				message.FunctionCall = &openai.FunctionCall{
					Name:      acc.Name,
					Arguments: acc.Arguments,
				}
				finishReason = openai.FinishReasonFunctionCall
				break
			}
		}
	}

	if len(toolCallAccumulator) > 0 {
		var toolCalls []openai.ToolCall
		for _, acc := range toolCallAccumulator {
			if acc != nil && acc.Complete {
				toolCalls = append(toolCalls, openai.ToolCall{
					ID:   acc.ID,
					Type: openai.ToolType(acc.Type),
					Function: openai.FunctionCall{
						Name:      acc.Function.Name,
						Arguments: acc.Function.Arguments,
					},
				})
			}
		}

		if len(toolCalls) > 0 {
			message.ToolCalls = toolCalls
			finishReason = openai.FinishReasonToolCalls
		}
	}

	choices := []openai.ChatCompletionChoice{
		{
			Index:        0,
			Message:      message,
			FinishReason: finishReason,
		},
	}

	promptTokens := c.estimateTokens(request.Messages)
	completionTokens := c.estimateTokens([]openai.ChatCompletionMessage{message})
	totalTokens := promptTokens + completionTokens

	return openai.ChatCompletionResponse{
		ID:      "",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: choices,
		Usage: openai.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
		},
	}
}

func (c *ChatCompletionClient) estimateTokens(messages []openai.ChatCompletionMessage) int {
	var allText strings.Builder

	for _, msg := range messages {
		allText.WriteString(msg.Content)
		allText.WriteString(" ")

		if msg.FunctionCall != nil {
			allText.WriteString(msg.FunctionCall.Name)
			allText.WriteString(" ")
			allText.WriteString(msg.FunctionCall.Arguments)
			allText.WriteString(" ")
		}

		for _, toolCall := range msg.ToolCalls {
			allText.WriteString(toolCall.ID)
			allText.WriteString(" ")
			allText.WriteString(toolCall.Function.Name)
			allText.WriteString(" ")
			allText.WriteString(toolCall.Function.Arguments)
			allText.WriteString(" ")
		}
	}

	normalized := strings.Join(strings.Fields(allText.String()), " ")
	words := strings.Fields(normalized)
	return len(words)
}

func (c *ChatCompletionClient) sendAsyncError(errChan chan<- error, err error) {
	if err == nil {
		return
	}

	select {
	case errChan <- err:
	default:
	}
}

func (c *ChatCompletionClient) BaseURL() string {
	return c.baseURL
}

func normalizeBaseURL(baseURL string) string {
	trimmed := strings.TrimSpace(baseURL)
	trimmed = strings.TrimRight(trimmed, "/")
	return trimmed
}

func statusCode(resp *resty.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode()
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SanitizeMessages removes invalid message parts that would cause validation errors
// with LLM providers. This includes:
// - Empty text parts (type: "text" without text field due to omitempty)
// - Empty image parts (type: "image_url" without URL)
func SanitizeMessages(messages []openai.ChatCompletionMessage) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, 0, len(messages))

	for _, msg := range messages {
		sanitizedMsg := msg

		// Sanitize MultiContent array if present
		if len(msg.MultiContent) > 0 {
			sanitizedParts := make([]openai.ChatMessagePart, 0, len(msg.MultiContent))
			for _, part := range msg.MultiContent {
				// Skip empty text parts (would serialize as {"type": "text"} without text field)
				if part.Type == openai.ChatMessagePartTypeText && part.Text == "" {
					continue
				}
				// Skip empty image parts (no URL)
				if part.Type == openai.ChatMessagePartTypeImageURL && (part.ImageURL == nil || part.ImageURL.URL == "") {
					continue
				}
				sanitizedParts = append(sanitizedParts, part)
			}

			// If all parts were filtered out, convert to a simple text message
			if len(sanitizedParts) == 0 && msg.Content != "" {
				sanitizedMsg.MultiContent = nil
			} else {
				sanitizedMsg.MultiContent = sanitizedParts
			}
		}

		result = append(result, sanitizedMsg)
	}

	return result
}
