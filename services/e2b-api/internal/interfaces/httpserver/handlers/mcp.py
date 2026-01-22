"""MCP static tools definitions for browser sandbox.

Tool naming convention:
- e2b_desktop_* : All hardcoded static tools
- e2b_* prefix : Added to dynamic tools from mcp-proxy

Note: e2b_sandbox_* lifecycle tools are defined in mcp-tools, not here.
"""

from internal.interfaces.httpserver.responses.mcp import (
    MCPTool,
    MCPToolInput,
)

# All static desktop tools (hardcoded in e2b-api)
STATIC_E2B_TOOLS: list[MCPTool] = [
    # Shell execution
    MCPTool(
        name="e2b_desktop_shell",
        description="Execute shell commands in the sandbox workspace. Returns stdout, stderr, and exit code. Runs in workspace directory (/home/user/workspace/{conversation_id}/).",
        inputSchema=MCPToolInput(
            properties={
                "conversation_id": {"type": "string", "description": "Conversation ID (workspace)"},
                "command": {"type": "string", "description": "Shell command to execute"},
                "timeout": {
                    "type": "integer",
                    "description": "Timeout in seconds (default: 60, max: 300)",
                    "default": 60,
                },
            },
            required=["conversation_id", "command"],
        ),
    ),
    # File operations
    MCPTool(
        name="e2b_desktop_file_read",
        description="Read file from workspace. Path is relative to workspace root.",
        inputSchema=MCPToolInput(
            properties={
                "conversation_id": {"type": "string", "description": "Conversation ID (workspace)"},
                "path": {"type": "string", "description": "Relative path within workspace"},
            },
            required=["conversation_id", "path"],
        ),
    ),
    MCPTool(
        name="e2b_desktop_file_write",
        description="Write content to a file in workspace. Creates parent directories if needed. Path is relative to workspace root.",
        inputSchema=MCPToolInput(
            properties={
                "conversation_id": {"type": "string", "description": "Conversation ID (workspace)"},
                "path": {"type": "string", "description": "Relative path within workspace"},
                "content": {"type": "string", "description": "Content to write"},
            },
            required=["conversation_id", "path", "content"],
        ),
    ),
    MCPTool(
        name="e2b_desktop_file_edit",
        description="Edit a file by replacing old_string with new_string. Use for making targeted changes to existing files.",
        inputSchema=MCPToolInput(
            properties={
                "conversation_id": {"type": "string", "description": "Conversation ID (workspace)"},
                "path": {"type": "string", "description": "Relative path within workspace"},
                "old_string": {"type": "string", "description": "The text to find and replace"},
                "new_string": {"type": "string", "description": "The text to replace with"},
            },
            required=["conversation_id", "path", "old_string", "new_string"],
        ),
    ),
    MCPTool(
        name="e2b_desktop_file_list",
        description="List files and directories in workspace. Path is relative to workspace root.",
        inputSchema=MCPToolInput(
            properties={
                "conversation_id": {"type": "string", "description": "Conversation ID (workspace)"},
                "path": {
                    "type": "string",
                    "description": "Relative directory path within workspace",
                    "default": ".",
                },
            },
            required=["conversation_id"],
        ),
    ),
    # Desktop interaction
    MCPTool(
        name="e2b_desktop_screenshot",
        description="Take a screenshot of the sandbox desktop. Returns base64-encoded PNG image.",
        inputSchema=MCPToolInput(properties={}, required=[]),
    ),
    MCPTool(
        name="e2b_desktop_click",
        description="Perform a mouse click at coordinates in the sandbox.",
        inputSchema=MCPToolInput(
            properties={
                "x": {"type": "integer", "description": "X coordinate"},
                "y": {"type": "integer", "description": "Y coordinate"},
                "button": {
                    "type": "string",
                    "enum": ["left", "right", "middle"],
                    "description": "Mouse button",
                    "default": "left",
                },
            },
            required=["x", "y"],
        ),
    ),
    MCPTool(
        name="e2b_desktop_type",
        description="Type text in the sandbox.",
        inputSchema=MCPToolInput(
            properties={
                "text": {"type": "string", "description": "Text to type"},
            },
            required=["text"],
        ),
    ),
    # Code execution
    MCPTool(
        name="e2b_desktop_code_execute",
        description="Execute Python or Node.js code in workspace. Code file and execution happen in workspace directory. Can access workspace files with relative paths.",
        inputSchema=MCPToolInput(
            properties={
                "conversation_id": {"type": "string", "description": "Conversation ID (workspace)"},
                "code": {"type": "string", "description": "Code to execute"},
                "language": {
                    "type": "string",
                    "enum": ["python", "nodejs"],
                    "description": "Programming language (default: python)",
                    "default": "python",
                },
                "timeout": {
                    "type": "integer",
                    "description": "Execution timeout in seconds (default: 60)",
                    "default": 60,
                },
            },
            required=["conversation_id", "code"],
        ),
    ),
    # Package installation
    MCPTool(
        name="e2b_desktop_packages",
        description="Install packages using pip (Python) or npm (Node.js) in the sandbox. Use this before running code that requires external libraries.",
        inputSchema=MCPToolInput(
            properties={
                "packages": {
                    "type": "array",
                    "items": {"type": "string"},
                    "description": "List of package names to install. Supports version specifiers.",
                },
                "package_manager": {
                    "type": "string",
                    "enum": ["pip", "npm"],
                    "description": "Package manager to use (default: pip)",
                    "default": "pip",
                },
            },
            required=["packages"],
        ),
    ),
]

# Set of static tool names for quick lookup
STATIC_TOOL_NAMES = {tool.name for tool in STATIC_E2B_TOOLS}
