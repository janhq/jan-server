# Prompt Orchestration

## Overview

The Prompt Orchestration system is a pipeline component within the LLM API service that dynamically composes and enhances prompts before they are sent to inference providers. It applies conditional modules based on context, user preferences, conversation history, and database-driven templates.

**Architecture Decision**: Prompt orchestration is implemented as a **processor within the LLM API service**, not as an isolated microservice. This gives you dynamic control at runtime to add memory, tools, templates, customize tone, and assemble final prompts automatically.

---

## What is a Prompt Orchestration Processor?

A **Prompt Orchestration Processor** is a processing layer within LLM API that:

1. Takes a user's raw input (before it reaches the inference engine)
2. Checks conditions (flags, context, user settings, memory, etc.)
3. Composes a final prompt programmatically by applying conditional modules
4. Passes that composed prompt to the inference provider (vLLM or remote)

The processor sits in the request pipeline within `llm-api`, between the HTTP handler and the inference provider client.

---

## Architecture

```
HTTP Request (POST /v1/chat/completions)
    ↓
Gin Handler (llm-api/internal/interfaces/httpserver/handlers/chathandler)
    ↓
1. Conversation Context Loading
   - Get or create conversation
   - Load conversation items
   - Load project instructions
    ↓
2. Memory Loading
   - Collect prompt memory from headers/metadata
   - Memory is handled in-process by the LLM API (gated by PROMPT_ORCHESTRATION_MEMORY)
    ↓
3. Provider Selection
   - Select provider model based on request
   - Load model catalog for context length
    ↓
4. Media Resolution
   - Resolve jan_* media placeholders
    ↓
5. Project Instruction Injection
   - Prepend project instruction as first system message
    ↓
6. Prompt Orchestration Processor
   - Build prompt context with preferences
   - Apply conditional modules in priority order
   - Return enhanced messages
    ↓
7. Context Trimming
   - Trim messages to fit model context length
    ↓
Inference Provider Client (internal/infrastructure/inference)
    ↓
vLLM or Remote Provider
```

### Package Structure

```
services/llm-api/internal/domain/prompt/
├── types.go                  # Core interfaces and types
├── modules.go                # Built-in module implementations
├── deep_research_module.go   # Deep Research module
├── processor.go              # Main processor implementation
└── processor_test.go         # Unit tests
```

---

## Features & Capabilities

### What the Processor Can Do

The processor can automatically attach optional modules as part of the LLM API request pipeline:

#### Deep Research

Inject specialized research prompts for comprehensive analysis when `deep_research: true` is set.

#### User Profile Personalization

Inject user profile settings including:

- Base style (concise, friendly, professional)
- Custom instructions
- Nickname, occupation, and personal context

#### Memory

If user enables memory, insert memory hints/preferences into prompt.

#### Tool Usage

Conditionally include instructions like:

- "use the retrieval tool when needed"
- "use the calculator tool if numbers appear"

#### Templates / Prompt Patterns

For example:

- Chain-of-Thought structure
- Code assistant guidance
- Output format
- "First think step-by-step, then answer"

#### Project Instructions

Inject project-specific instructions with highest priority.

#### Timing Context

Add current date and AI assistant introduction.

---

## Built-in Modules

The processor includes several built-in modules that are automatically applied based on context:

### -20. Deep Research Module (Conditional)

- **Purpose**: Injects comprehensive research prompts for deep analysis
- **Activation**: Enabled when `deep_research: true` in preferences
- **Adds**: Research methodology and comprehensive analysis instructions
- **Priority**: -20 (runs before all other modules)

### -15. Timing Module (Always Active)

- **Purpose**: Ensures a base system prompt with current date is present
- **Activation**: Always registered when prompt orchestration is enabled
- **Adds**: AI assistant intro and current date to the system prompt
- **Priority**: -15

### -10. Project Instruction Module (Conditional)

- **Purpose**: Injects project-specific instructions with highest priority
- **Activation**: When conversation has a linked project with instructions
- **Adds**: Project instructions as first system message with priority note
- **Priority**: -10

### 5. User Profile Module (Conditional)

- **Purpose**: Injects user profile personalization settings
- **Activation**: When user has profile settings (base style, custom instructions, nickname, etc.)
- **Adds**: Style preferences, custom instructions, and user context
- **Priority**: 5

### 10. Memory Module (Optional)

- **Purpose**: Injects user-specific memory/preferences into prompts
- **Activation**: Enabled via `PROMPT_ORCHESTRATION_MEMORY=true` and memory items present
- **Adds**: Memory hints stitched into the system prompt
- **Priority**: 10

### 20. Tool Instructions Module (Optional)

- **Purpose**: Adds instructions for tool usage
- **Activation**: `PROMPT_ORCHESTRATION_TOOLS=true` and tools present in request
- **Adds**: Tool selection and usage guidelines
- **Priority**: 20

### 30. Code Assistant Module (Template-Gated)

- **Purpose**: Enhances prompts for code-related questions
- **Activation**: `PROMPT_ORCHESTRATION_TEMPLATES=true` and code keywords detected
- **Adds**: Code formatting guidelines, best practices, error handling tips
- **Priority**: 30

### 40. Chain-of-Thought Module (Template-Gated)

- **Purpose**: Encourages step-by-step reasoning for complex questions
- **Activation**: `PROMPT_ORCHESTRATION_TEMPLATES=true` and reasoning signals detected
- **Adds**: Instructions to break down problems and think systematically
- **Priority**: 40

---

## Configuration

### Environment Variables

| Variable                         | Default | Description                                          |
| -------------------------------- | ------- | ---------------------------------------------------- |
| `PROMPT_ORCHESTRATION_ENABLED`   | `false` | Enable/disable the processor                         |
| `PROMPT_ORCHESTRATION_MEMORY`    | `false` | Enable memory injection                              |
| `PROMPT_ORCHESTRATION_TEMPLATES` | `false` | Enable template-based prompts (CoT + code assistant) |
| `PROMPT_ORCHESTRATION_TOOLS`     | `false` | Enable tool usage instructions                       |

### YAML Configuration

In `config/defaults.yaml`:

```yaml
services:
  llm_api:
    prompt_orchestration:
      enabled: false
      enable_memory: false
      enable_templates: false
      enable_tools: false
```

### Wire Integration

The processor is integrated via dependency injection in `services/llm-api/internal/domain/provider.go`:

```go
// ProvidePromptProcessor creates the prompt processor with all modules including Deep Research
func ProvidePromptProcessor(
	config prompt.ProcessorConfig,
	log zerolog.Logger,
	templateService *prompttemplate.Service,
) *prompt.ProcessorImpl {
	processor := prompt.NewProcessorWithTemplateService(config, log, templateService)

	// Register Deep Research module if prompt orchestration is enabled
	if config.Enabled && templateService != nil {
		processor.RegisterModule(prompt.NewDeepResearchModule(templateService))
	}

	return processor
}
```

---

## Implementation Details

### Module Interface

Each module implements the `Module` interface:

```go
type Module interface {
    Name() string
    ShouldApply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) bool
    Apply(ctx context.Context, promptCtx *Context, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error)
}
```

### Context Structure

The prompt context contains all information needed for module decisions:

```go
type Context struct {
    UserID             uint
    ConversationID     string
    Language           string
    Preferences        map[string]interface{}
    Memory             []string
    ProjectInstruction string
    AppliedModules     []string
    Profile            *usersettings.ProfileSettings
}
```

### Processing Flow

1. **Context Building**: Create a `prompt.Context` with user ID, conversation ID, preferences, memory, project instruction, and user profile
2. **Module Evaluation**: Each registered module checks if it should apply via `ShouldApply()`
3. **Module Application**: Applicable modules modify messages via `Apply()` in priority order
4. **Result**: Enhanced messages are passed to the inference provider

### Module Priority System

Modules are executed in priority order to ensure correct composition:

- **Priority -20**: Deep Research Module (comprehensive research prompts)
- **Priority -15**: Timing Module (creates base system prompt with date)
- **Priority -10**: Project Instruction Module (highest priority instructions)
- **Priority 5**: User Profile Module (personalization settings)
- **Priority 10**: Memory Module (adds user context)
- **Priority 20**: Tool Instructions (adds tool capabilities)
- **Priority 30**: Code Assistant (adds code-specific guidance)
- **Priority 40**: Chain-of-Thought (adds reasoning structure)

---

## Usage in Chat Handler

The processor is integrated into the chat completion flow in `chat_handler.go`:

```go
// Apply prompt orchestration (if enabled)
if h.promptProcessor != nil {
    observability.AddSpanEvent(ctx, "processing_prompts")

    preferences := make(map[string]interface{})
    if len(request.Tools) > 0 || request.ToolChoice != nil {
        preferences["use_tools"] = true
    }
    if persona := strings.TrimSpace(reqCtx.GetHeader("X-Prompt-Persona")); persona != "" {
        preferences["persona"] = persona
    }
    if persona := strings.TrimSpace(reqCtx.Query("persona")); persona != "" {
        preferences["persona"] = persona
    }

    // Pass deep_research flag to prompt orchestration
    if request.DeepResearch != nil && *request.DeepResearch {
        preferences["deep_research"] = true
    }

    var profileSettings *usersettings.ProfileSettings
    if userSettings != nil {
        profileSettings = &userSettings.ProfileSettings
    }

    promptCtx := &prompt.Context{
        UserID:             userID,
        ConversationID:     conversationID,
        Language:           strings.TrimSpace(reqCtx.GetHeader("Accept-Language")),
        Preferences:        preferences,
        Memory:             loadedMemory,
        ProjectInstruction: projectInstruction,
        Profile:            profileSettings,
    }

    processedMessages, processErr := h.promptProcessor.Process(ctx, promptCtx, request.Messages)
    if processErr != nil {
        // Continue with original messages
    } else {
        request.Messages = processedMessages
        if len(promptCtx.AppliedModules) > 0 {
            reqCtx.Header("X-Applied-Prompt-Modules", strings.Join(promptCtx.AppliedModules, ","))
        }
        observability.AddSpanEvent(ctx, "prompts_processed")
    }
}
```

---

## Template Service Integration

Modules can load prompts from the database via the `prompttemplate.Service`:

```go
// Try to fetch template from database and render with variables
if m.templateService != nil {
    template, err := m.templateService.GetByKey(ctx, prompttemplate.TemplateKeyTiming)
    if err == nil && template != nil && template.IsActive {
        rendered, renderErr := m.templateService.RenderTemplate(ctx, prompttemplate.TemplateKeyTiming, map[string]any{
            "CurrentDate": currentDate,
        })
        if renderErr == nil {
            timingText = rendered
        }
    }
}

// Fallback to hardcoded text if template not loaded
if timingText == "" {
    timingText = "You are Jan, a helpful AI assistant..."
}
```

### Available Template Keys

| Template Key        | Module                 | Variables                                                                   |
| ------------------- | ---------------------- | --------------------------------------------------------------------------- |
| `timing`            | TimingModule           | `CurrentDate`                                                               |
| `user_profile`      | UserProfileModule      | `BaseStyle`, `CustomInstructions`, `NickName`, `Occupation`, `MoreAboutYou` |
| `memory`            | MemoryModule           | `MemoryItems`                                                               |
| `tool_instructions` | ToolInstructionsModule | `ToolDescriptions`                                                          |
| `code_assistant`    | CodeAssistantModule    | (none)                                                                      |
| `chain_of_thought`  | ChainOfThoughtModule   | (none)                                                                      |
| `deep_research`     | DeepResearchModule     | (none)                                                                      |

---

## Example Transformations

### Before Processing

```json
{
  "messages": [
    { "role": "user", "content": "How do I implement binary search in Go?" }
  ]
}
```

### After Processing

_With Timing + User Profile + Code Assistant modules applied:_

```json
{
  "messages": [
    {
      "role": "system",
      "content": "You are Jan, a helpful AI assistant. Jan is trained by Menlo Research (https://www.menlo.ai).\nToday is: December 16, 2025.\nAlways treat this as the current date.\n\nUser-level settings are preferences for style and context. If they ever conflict with explicit project or system instructions, always follow the project or system instructions.\n\nUse a friendly, warm, and encouraging tone while staying helpful.\n\nWhen providing code assistance:\n1. Provide clear, well-commented code.\n2. Explain your approach and reasoning.\n3. Include error handling where appropriate.\n4. Follow best practices and conventions.\n5. Suggest testing approaches when relevant.\n6. Respect project instructions and user constraints; never violate them to simplify code."
    },
    { "role": "user", "content": "How do I implement binary search in Go?" }
  ]
}
```

### With Project Instructions

When a conversation is linked to a project with instructions:

```json
{
  "messages": [
    {
      "role": "system",
      "content": "Always respond in JSON format. Use TypeScript conventions.\n\nProject priority: These project-specific instructions have the highest priority. If any user settings, style preferences, or other guidance conflict with these project instructions, you must follow the project instructions."
    },
    {
      "role": "system",
      "content": "You are Jan, a helpful AI assistant...\n\nUse a friendly, warm, and encouraging tone..."
    },
    {
      "role": "user",
      "content": "Create a function to validate email addresses"
    }
  ]
}
```

---

## Disabling Modules

Users can disable specific modules via preferences:

```go
promptCtx := &prompt.Context{
    Preferences: map[string]interface{}{
        "disable_modules": []string{"chain_of_thought", "code_assistant"},
    },
}
```

Or via the helper function:

```go
promptCtx = prompt.WithDisabledModules(promptCtx, []string{"memory"})
```

Supported formats for `disable_modules`:

- Comma-separated string: `"chain_of_thought,code_assistant"`
- String slice: `[]string{"chain_of_thought", "code_assistant"}`
- Interface slice: `[]interface{}{"chain_of_thought", "code_assistant"}`

---

## Observability

The processor emits:

- **OTEL events**: `processing_prompts`, `prompts_processed`
- **Logs**: Applied module list, processing errors, module priority order
- **HTTP header**: `X-Applied-Prompt-Modules` (comma-separated) for debugging

Example log output:

```json
{
  "level": "debug",
  "component": "prompt-processor",
  "conversation_id": "conv-123",
  "applied_modules": ["timing", "user_profile", "memory", "code_assistant"],
  "message": "applied prompt orchestration modules"
}
```

---

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
- Module priority ordering
- Template service integration

---

## Future Enhancements

Potential additions to the processor:

1. **Template Library**: Pre-built templates for common tasks (writing, analysis, translation)
2. **Dynamic Persona**: Adjust assistant personality based on context
3. **Language Detection**: Automatically adapt to user's language
4. **Safety Filters**: Add content moderation and safety rules
5. **A/B Testing**: Compare different prompt strategies
6. **Custom Module Registry**: Allow users to register custom modules
7. **Module Composition Rules**: Define dependencies and conflicts between modules
8. **Prompt Versioning**: Track and version prompt templates
9. **Performance Optimization**: Cache compiled prompts for common scenarios

---

## Related Documentation

- [Data Flow Reference](../architecture/data-flow.md)
- [LLM API Documentation](../api/llm-api/README.md)
- [Development Guide](./development.md)
- [Testing Guide](./testing.md)

---

## Troubleshooting

### Modules Not Applying

**Check:**

1. Is `PROMPT_ORCHESTRATION_ENABLED=true`?
2. Are specific module flags enabled (`MEMORY`, `TEMPLATES`, `TOOLS`)?
3. Does the module's `ShouldApply()` logic match your request?
4. Check logs for `X-Applied-Prompt-Modules` header

### Module Order Issues

**Solution:**
Modules execute in priority order (-20, -15, -10, 5, 10, 20, 30, 40). Deep Research runs first, then Timing, then Project Instructions.

### Memory Not Loading

**Check:**

1. Is memory provided via `X-Prompt-Memory` header or conversation metadata?
2. Is `promptCtx.Memory` populated with items?
3. Is `PROMPT_ORCHESTRATION_MEMORY=true`? (Memory is composed in-process; there is no separate memory service.)

### User Profile Not Applying

**Check:**

1. Does the user have profile settings configured?
2. Is at least one profile field non-empty (BaseStyle, CustomInstructions, NickName, etc.)?
3. Is the `user_profile` module not in the disabled list?

### Template Not Loading from Database

**Check:**

1. Is the template service properly initialized?
2. Is the template active (`is_active: true`)?
3. Check logs for template loading errors
4. Fallback prompts will be used if template fails

### Performance Concerns

**Optimization:**

- Modules are sorted once during processor initialization
- Each module only applies if `ShouldApply()` returns true
- Template service caches templates
- Consider disabling unused modules via environment variables
