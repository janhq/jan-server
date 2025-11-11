# Memory Tooling TODO

This document captures the "Jan Has Memory" feature request so we can design, prioritize, and build shared persistent tooling in the repo.

## 1. Memory Tool Categories

- **User memory** - Long-term key/value store for personal facts, preferences, habits, schedules that improve personalization across sessions. These items belong to a user and are retrieved whenever that user starts a chat. Example facts: `preferred response style`, `timezone`, `dietary restrictions`.
- **Project memory** - Workspace-aware knowledge graph of project decisions, goals, deliverables, and tooling assumptions that everyone on the project can query. Stored per `project_id`, backed by vector embeddings plus metadata, and queried when a user switches to that project context.
- **Conversation memory** - Short-term context that rolls with the current chat session. Maintained in hot cache (Redis or similar) and optionally summarized to MemSvc for durable recall. Includes dialogue turns, useful tool outputs, and in-flight tasks.

## 2. Memory Ingestion Feed Rules

### User Memory Ingestion

1. **Sources**
   - Explicit style commands such as "remember this", friendly profile forms, or settings pages.
   - System or message heuristics that detect personal preferences from stage instructions or repeated tool usage.
   - Confirmations or corrections where a user states identity, habits, or stable schedules.
2. **Selection logic**
   - Default to "selected chats only"; offer a per-conversation toggle named `Allow saving to User Memory`.
   - Score candidate facts:
     - +2 when explicitly prefixed with "remember" or "store this".
     - +1 when the same preference appears in two distinct conversations.
     - -1 when a later message contradicts the fact.
   - Persist the fact when the score reaches 2 or higher.
3. **Storage and access**
   - Use a key/value store (or embedding metadata table) keyed by user ID.
   - Retrieve before each session so the system can tailor greetings and prompts.

### Conversation Memory Ingestion

1. **Sources**
   - Every message in the active conversation, including system messages and relevant tool outputs.
2. **Strategy**
   - Keep a hot window of N messages in Redis for prompt injection.
   - Periodically run `summarize_window` (based on message count or elapsed time) to emit:
     - `dialogue_summary`
     - `open_tasks`
     - `entities`
   - Store the summary in the MemSvc namespace `conversation`.
   - Detect cues such as "done" or "ship it" and snapshot milestones when they appear.

### Project Memory Ingestion

1. **Sources**
   - All conversations where `conversation.project_id == project_id`, plus finalized documents, specs, PRDs, notes, PDFs, code snippets.
2. **Selection rules**
   - Ingest only finalized artifacts by default.
   - Within conversations, ingest statements tagged as `decision`, `assumption`, `risk`, `todo`, or `metric`.
3. **File ingestion**
   - Chunk attachments using document-aware heuristics (headings, code blocks, tables).
   - Generate embeddings per chunk, store the chunk text in `memory_items` and its vector in `memory_vectors`, all linked to `project_id`.
4. **Conversation-to-project promotion**
   - Provide a UI action like "Promote to project memory" or trigger automatically on confirmed decisions.
   - Emit `project_fact` entries that capture the reasoning and rationale behind each promoted decision.

## 3. Implementation TODOs

1. **Infrastructure and storage**
   - Define schemas for `UserMemoryItem`, `ConversationSummary`, `ProjectFact`, and `ProjectChunk`.
   - Integrate a vector store for project chunks.
   - Add a Redis-backed window for conversation context.
2. **Ingestion pipelines**
   - Build the user memory scoring pipeline with the heuristics above plus acceptance toggles.
   - Schedule a `summarize_window` worker that runs on interval or message thresholds.
   - Assemble project ingestion for attachments and promoted conversation facts.
3. **APIs and UX**
   - Expose endpoints that toggle user memory consent per conversation.
   - Allow promotion of conversation facts to project memory and flag key decisions.
   - Surface summaries, open tasks, and entities inside the chat interface.
4. **Governance**
   - Document what qualifies for user versus project memory.
   - Respect privacy by asking for consent and offering memory deletion.
   - Log read/write operations to support auditing.

## 4. Next Steps

1. Prioritize ingestion workstreams (user memory scoring plus conversation summarizer).
2. Wire Redis/embedding data stores into existing MCP tooling.
3. Draft UI/UX flows for toggles and summaries in docs or the README.
4. Validate the feature with sample data: capture user preferences and project facts and confirm they surface under the rules above.
