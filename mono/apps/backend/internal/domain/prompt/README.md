# Prompt Orchestration Processor

## Overview

The Prompt Orchestration Processor is a pipeline component within the LLM API service that dynamically composes and enhances prompts before they are sent to inference providers. It applies conditional modules based on context, user preferences, and conversation history.

## Architecture

```
HTTP Request (POST /v1/chat/completions)
    ↓
Gin Handler
    ↓
Chat Handler
    ↓
Prompt Orchestration Processor ← YOU ARE HERE
    - Check context & user preferences
    - Apply conditional modules
    - Assemble final prompts
    ↓
Inference Provider Client
    ↓
vLLM or Remote Provider
```

## Features

### Conditional Modules

The processor includes several built-in modules that are automatically applied based on context:

#### 1. **Memory Module** (Optional)

- **Purpose**: Injects user-specific memory/preferences into prompts
- **Activation**: Enabled via `PROMPT_ORCHESTRATION_MEMORY=true`
- **Example**: Adds "User prefers concise answers" to system prompt

#### 2. **Code Assistant Module** (Always Active)

- **Purpose**: Enhances prompts for code-related questions
- **Activation**: Automatically detects code keywords (function, implement, debug, etc.)
- **Adds**: Code formatting guidelines, best practices, error handling tips

#### 3. **Chain-of-Thought Module** (Always Active)

- **Purpose**: Encourages step-by-step reasoning for complex questions
- **Activation**: Detects questions with reasoning keywords (why, how, explain, analyze)
- **Adds**: Instructions to break down problems and think systematically

#### 4. **Tool Instructions Module** (Optional)

- **Purpose**: Adds instructions for tool usage
- **Activation**: Enabled via `PROMPT_ORCHESTRATION_TOOLS=true` and user preferences
- **Adds**: Tool selection and usage guidelines

## Configuration

### Environment Variables

| Variable                         | Default | Description                    |
| -------------------------------- | ------- | ------------------------------ |
| `PROMPT_ORCHESTRATION_ENABLED`   | `true`  | Enable/disable the processor   |
| `PROMPT_ORCHESTRATION_MEMORY`    | `false` | Enable memory injection        |
| `PROMPT_ORCHESTRATION_TEMPLATES` | `true`  | Enable template-based prompts  |
| `PROMPT_ORCHESTRATION_TOOLS`     | `false` | Enable tool usage instructions |

### YAML Configuration

In `config/defaults.yaml`:

```yaml
services:
  llm_api:
    prompt_orchestration:
      enabled: true
      enable_memory: false
      enable_templates: true
      enable_tools: false
```

## Implementation Details

### Package Structure

```
services/llm-api/internal/domain/prompt/
├── types.go          # Core interfaces and types
├── modules.go        # Built-in module implementations
├── processor.go      # Main processor implementation
└── processor_test.go # Comprehensive tests
```

### Module Interface

Each module implements the `Module` interface:

```go
type Module interface {
    Name() string
    ShouldApply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) bool
    Apply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error)
}
```

### Processing Flow

1. **Context Building**: Create a `prompt.Context` with user ID, conversation ID, preferences, and memory
2. **Module Evaluation**: Each registered module checks if it should apply via `ShouldApply()`
3. **Module Application**: Applicable modules modify messages via `Apply()`
4. **Result**: Enhanced messages are passed to the inference provider

## Usage Example

The processor is automatically integrated into the chat completion flow:

```go
// In ChatHandler.CreateChatCompletion()
promptCtx := &prompt.Context{
    UserID:         userID,
    ConversationID: conversationID,
    Preferences:    make(map[string]interface{}),
    Memory:         []string{}, // Load from user memory store
}

processedMessages, err := h.promptProcessor.Process(ctx, promptCtx, request.Messages)
if err != nil {
    // Log and continue with original messages
    log.Warn().Err(err).Msg("prompt processing failed")
} else {
    request.Messages = processedMessages
}
```

## Example Transformations

### Before Processing

```json
{
  "messages": [
    { "role": "user", "content": "How do I implement binary search in Go?" }
  ]
}
```

### After Processing (Code Assistant + Memory modules applied)

```json
{
  "messages": [
    {
      "role": "system",
      "content": "You are Jan, a helpful AI assistant.\n\nUse the following personal memory for this user:\n- User prefers detailed code examples\n- User is learning Go\n\nWhen providing code assistance:\n1. Provide clear, well-commented code\n2. Explain your approach and reasoning\n3. Include error handling where appropriate\n4. Follow best practices and conventions\n5. Suggest testing approaches when relevant"
    },
    { "role": "user", "content": "How do I implement binary search in Go?" }
  ]
}
```

## Testing

Run the test suite:

```bash
cd services/llm-api
go test ./internal/domain/prompt/... -v
```

Tests cover:

- Individual module behavior
- Module conditional logic
- Processor integration
- Configuration handling

## Observability

The processor emits OpenTelemetry events:

- `processing_prompts`: When processing starts
- `prompts_processed`: When processing completes successfully

Logs include:

- Applied modules list
- Processing errors (non-fatal)
- Conversation and user context

## Future Enhancements

Potential additions to the processor:

1. **Template Library**: Pre-built templates for common tasks (writing, analysis, translation)
2. **User Memory Store**: Persistent storage for user preferences and memory
3. **Dynamic Persona**: Adjust assistant personality based on context
4. **Language Detection**: Automatically adapt to user's language
5. **Safety Filters**: Add content moderation and safety rules
6. **A/B Testing**: Compare different prompt strategies

## Related Documentation

- [Prompt Orchestration Design](../../../docs/todo/prompt-orchestration-todo.md)
- [Data Flow Reference](../../../docs/architecture/data-flow.md)
- [LLM API Documentation](../../../docs/api/llm-api/README.md)
