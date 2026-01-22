"""User sandbox HTTP handlers - User-centric operations.

Handlers are thin - they delegate to domain services and let domain
exceptions propagate to the global exception handler.
"""

import base64
import json
import logging
import os
import time
from typing import Any

from internal.domain.exceptions import SandboxNotFoundError
from internal.domain.sandbox.entity import SandboxRecord, SandboxStatus, Workspace
from internal.domain.sandbox.gateway import SandboxGateway
from internal.domain.sandbox.service import UserSandboxService
from internal.infrastructure.mcp.session_store import MCPSessionStore, ToolInfo
from internal.interfaces.httpserver.handlers.mcp import STATIC_E2B_TOOLS, STATIC_TOOL_NAMES
from internal.interfaces.httpserver.requests.sandbox import (
    ExtendTimeoutRequest,
    StartSandboxRequest,
)
from internal.interfaces.httpserver.responses.sandbox import SandboxHealthResponse
from internal.interfaces.httpserver.responses.workspace import (
    SandboxRecordResponse,
    WorkspaceResponse,
    WorkspaceWithSandboxResponse,
)
from internal.interfaces.httpserver.responses.mcp import (
    MCPError,
    MCPInitializeResult,
    MCPRequest,
    MCPResponse,
    MCPTool,
    MCPToolCallResult,
    MCPToolInput,
    MCPToolsListResult,
)

logger = logging.getLogger(__name__)


def _sandbox_to_response(sandbox: SandboxRecord) -> SandboxRecordResponse:
    """Convert domain entity to response."""
    return SandboxRecordResponse(
        public_id=sandbox.public_id,
        user_id=sandbox.user_id,
        e2b_sandbox_id=sandbox.e2b_sandbox_id,
        status=sandbox.status.value,
        view_url=sandbox.view_url,
        control_url=sandbox.control_url,
        sandbox_url=sandbox.sandbox_url,
        started_at=sandbox.started_at,
        timeout_at=sandbox.timeout_at,
        paused_at=sandbox.paused_at,
        pause_expires_at=sandbox.pause_expires_at,
        error_message=sandbox.error_message,
        created_at=sandbox.created_at,
    )


def _workspace_to_response(workspace: Workspace) -> WorkspaceResponse:
    """Convert domain entity to response."""
    return WorkspaceResponse(
        public_id=workspace.public_id,
        conversation_id=workspace.conversation_id,
        workspace_path=workspace.workspace_path,
        created_at=workspace.created_at,
    )


class UserSandboxHandler:
    """Handler for user-centric sandbox HTTP endpoints.

    Handlers are thin - they delegate to domain services and let domain
    exceptions propagate to the global exception handler.
    """

    def __init__(
        self,
        service: UserSandboxService,
        gateway: SandboxGateway,
        session_store: MCPSessionStore,
    ):
        self.service = service
        self._gateway = gateway
        self._session_store = session_store

    def _resolve_workspace_path(self, conversation_id: str, relative_path: str) -> str:
        """Resolve relative path to absolute workspace path.

        Args:
            conversation_id: The conversation/workspace ID
            relative_path: Path relative to workspace root

        Returns:
            Absolute path within workspace

        Raises:
            ValueError: If path escapes workspace (path traversal attempt)
        """
        workspace_root = f"/home/user/workspace/{conversation_id}"

        # Normalize and check for traversal
        normalized = os.path.normpath(os.path.join(workspace_root, relative_path))
        if not normalized.startswith(workspace_root):
            raise ValueError(f"Path '{relative_path}' escapes workspace")

        return normalized

    async def get_sandbox(self, user_id: str) -> SandboxRecordResponse:
        """Get sandbox for a user."""
        sandbox = await self.service.get_sandbox_by_user(user_id)
        if sandbox is None:
            raise SandboxNotFoundError(user_id)
        return _sandbox_to_response(sandbox)

    async def start_sandbox(
        self, user_id: str, request: StartSandboxRequest | None = None
    ) -> SandboxRecordResponse:
        """Start sandbox for a user."""
        timeout = request.timeout if request else None
        sandbox = await self.service.start_sandbox(user_id, timeout=timeout)
        return _sandbox_to_response(sandbox)

    async def pause_sandbox(self, user_id: str) -> SandboxRecordResponse:
        """Pause sandbox for a user."""
        sandbox = await self.service.pause_sandbox(user_id)
        return _sandbox_to_response(sandbox)

    async def extend_timeout(
        self, user_id: str, request: ExtendTimeoutRequest
    ) -> SandboxRecordResponse:
        """Extend timeout for a user's sandbox."""
        sandbox = await self.service.extend_timeout(user_id, request.additional_seconds)
        return _sandbox_to_response(sandbox)

    async def delete_sandbox(self, user_id: str) -> SandboxRecordResponse:
        """Delete sandbox for a user (stops E2B and removes DB record)."""
        sandbox = await self.service.delete_sandbox(user_id)
        return _sandbox_to_response(sandbox)

    async def get_or_create_workspace(
        self, user_id: str, conversation_id: str
    ) -> WorkspaceWithSandboxResponse:
        """Get or create workspace for a conversation."""
        sandbox, workspace = await self.service.get_or_create_workspace(
            user_id, conversation_id
        )
        return WorkspaceWithSandboxResponse(
            sandbox=_sandbox_to_response(sandbox),
            workspace=_workspace_to_response(workspace),
        )

    async def sandbox_health(self, user_id: str) -> SandboxHealthResponse:
        """Check health of a user's sandbox by pinging it."""
        sandbox = await self.service.get_sandbox_by_user(user_id)
        if sandbox is None:
            raise SandboxNotFoundError(user_id)

        if sandbox.status != SandboxStatus.RUNNING or not sandbox.e2b_sandbox_id:
            return SandboxHealthResponse(
                sandbox_id=sandbox.public_id,
                status=sandbox.status.value,
                healthy=False,
                error="Sandbox is not running",
            )

        # Ping sandbox by running a simple command
        start_time = time.time()
        healthy = False
        error_msg: str | None = None
        ping_ms = 0

        try:
            result = self._gateway.run_command(sandbox.e2b_sandbox_id, "echo ping", timeout=10)
            ping_ms = int((time.time() - start_time) * 1000)

            if result.is_success():
                healthy = True
            else:
                error_msg = result.error or result.stderr
        except Exception as e:
            ping_ms = int((time.time() - start_time) * 1000)
            error_msg = str(e)

        # Sync DB status if sandbox is not healthy but DB shows running
        if not healthy and sandbox.status == SandboxStatus.RUNNING:
            sandbox = await self.service.sync_sandbox_state(sandbox)

        return SandboxHealthResponse(
            sandbox_id=sandbox.public_id,
            status=sandbox.status.value,
            healthy=healthy,
            ping_ms=ping_ms if ping_ms > 0 else None,
            error=error_msg,
        )

    async def handle_mcp_request(self, user_id: str, request: MCPRequest) -> MCPResponse:
        """Handle an MCP JSON-RPC 2.0 request for a user's sandbox."""
        try:
            # Get sandbox for user
            sandbox = await self.service.get_sandbox_by_user(user_id)
            if sandbox is None:
                return self._error_response(request.id, -32001, f"No sandbox found for user: {user_id}")

            # Validate actual E2B state before MCP operation
            sandbox = await self.service.sync_sandbox_state(sandbox)

            if sandbox.status != SandboxStatus.RUNNING:
                return self._error_response(request.id, -32001, f"Sandbox not running (status: {sandbox.status.value})")

            # Route by method
            if request.method == "initialize":
                return self._handle_initialize(request)
            elif request.method == "tools/list":
                return await self._handle_tools_list(request, sandbox.e2b_sandbox_id)
            elif request.method == "tools/call":
                return await self._handle_tools_call(request, sandbox.e2b_sandbox_id)
            else:
                return self._error_response(request.id, -32601, f"Method not found: {request.method}")
        except Exception as e:
            logger.error(f"MCP request error: {e}")
            return self._error_response(request.id, -32603, str(e))

    def _handle_initialize(self, request: MCPRequest) -> MCPResponse:
        """Handle MCP initialize request."""
        return MCPResponse(
            id=request.id,
            result=MCPInitializeResult().model_dump(),
        )

    async def _handle_tools_list(
        self, request: MCPRequest, e2b_sandbox_id: str | None
    ) -> MCPResponse:
        """Handle MCP tools/list request."""
        tools = list(STATIC_E2B_TOOLS)

        # Try to get dynamic tools if sandbox is running
        if e2b_sandbox_id:
            dynamic_tools = await self._get_dynamic_tools(e2b_sandbox_id)
            tools.extend(dynamic_tools)

        return MCPResponse(
            id=request.id,
            result=MCPToolsListResult(tools=tools).model_dump(),
        )

    async def _handle_tools_call(
        self, request: MCPRequest, e2b_sandbox_id: str | None
    ) -> MCPResponse:
        """Handle MCP tools/call request."""
        params = request.params or {}
        tool_name = params.get("name", "")
        arguments = params.get("arguments", {})

        if not tool_name:
            return self._error_response(request.id, -32602, "Missing tool name")

        if not e2b_sandbox_id:
            return self._error_response(request.id, -32001, "Sandbox not running")

        # Check if it's a static E2B tool
        if tool_name in STATIC_TOOL_NAMES:
            return await self._execute_static_tool(request.id, e2b_sandbox_id, tool_name, arguments)

        # Otherwise, proxy to sandbox's mcp-proxy (dynamic tool)
        return await self._proxy_dynamic_tool_call(request, e2b_sandbox_id)

    async def _execute_static_tool(
        self,
        request_id: int | str | None,
        e2b_sandbox_id: str,
        tool_name: str,
        arguments: dict[str, Any],
    ) -> MCPResponse:
        """Execute a static sandbox tool."""
        try:
            result_content: list[dict[str, Any]] = []

            if tool_name == "e2b_desktop_shell":
                conversation_id = arguments.get("conversation_id")
                command = arguments.get("command", "")
                timeout = arguments.get("timeout", 60)
                if not conversation_id:
                    return self._error_response(request_id, -32602, "conversation_id required")
                # Run command in workspace directory
                cwd = f"/home/user/workspace/{conversation_id}"
                result = self._gateway.run_command(e2b_sandbox_id, command, timeout, cwd=cwd)
                result_content = [
                    {
                        "type": "text",
                        "text": f"stdout: {result.stdout or ''}\nstderr: {result.stderr or ''}\nexit_code: {result.exit_code}",
                    }
                ]
                if result.error:
                    return self._error_response(request_id, -32000, result.error)

            elif tool_name == "e2b_desktop_file_read":
                conversation_id = arguments.get("conversation_id")
                relative_path = arguments.get("path", "")
                if not conversation_id:
                    return self._error_response(request_id, -32602, "conversation_id required")
                try:
                    abs_path = self._resolve_workspace_path(conversation_id, relative_path)
                except ValueError as e:
                    return self._error_response(request_id, -32602, str(e))
                result = self._gateway.read_file(e2b_sandbox_id, abs_path)
                # Decode base64 content to text
                try:
                    decoded_content = base64.b64decode(result.content).decode("utf-8")
                except Exception:
                    decoded_content = result.content  # Return as-is if decode fails
                result_content = [{"type": "text", "text": decoded_content}]

            elif tool_name == "e2b_desktop_file_write":
                conversation_id = arguments.get("conversation_id")
                relative_path = arguments.get("path", "")
                content = arguments.get("content", "")
                if not conversation_id:
                    return self._error_response(request_id, -32602, "conversation_id required")
                try:
                    abs_path = self._resolve_workspace_path(conversation_id, relative_path)
                except ValueError as e:
                    return self._error_response(request_id, -32602, str(e))
                result = self._gateway.write_file(e2b_sandbox_id, abs_path, content)
                result_content = [
                    {"type": "text", "text": f"Written {result.bytes_written} bytes to {relative_path}"}
                ]

            elif tool_name == "e2b_desktop_file_edit":
                conversation_id = arguments.get("conversation_id")
                relative_path = arguments.get("path", "")
                old_string = arguments.get("old_string", "")
                new_string = arguments.get("new_string", "")
                if not conversation_id:
                    return self._error_response(request_id, -32602, "conversation_id required")
                if not old_string:
                    return self._error_response(request_id, -32602, "old_string required")
                try:
                    abs_path = self._resolve_workspace_path(conversation_id, relative_path)
                except ValueError as e:
                    return self._error_response(request_id, -32602, str(e))
                # Read current content
                read_result = self._gateway.read_file(e2b_sandbox_id, abs_path)
                try:
                    current_content = base64.b64decode(read_result.content).decode("utf-8")
                except Exception:
                    return self._error_response(request_id, -32000, "Failed to read file for editing")
                # Check if old_string exists
                if old_string not in current_content:
                    return self._error_response(
                        request_id, -32000, f"old_string not found in {relative_path}"
                    )
                # Replace and write back
                new_content = current_content.replace(old_string, new_string, 1)
                self._gateway.write_file(e2b_sandbox_id, abs_path, new_content)
                result_content = [
                    {"type": "text", "text": f"Edited {relative_path}: replaced 1 occurrence"}
                ]

            elif tool_name == "e2b_desktop_file_list":
                conversation_id = arguments.get("conversation_id")
                relative_path = arguments.get("path", ".")
                if not conversation_id:
                    return self._error_response(request_id, -32602, "conversation_id required")
                try:
                    abs_path = self._resolve_workspace_path(conversation_id, relative_path)
                except ValueError as e:
                    return self._error_response(request_id, -32602, str(e))
                files = self._gateway.list_files(e2b_sandbox_id, abs_path)
                file_list = "\n".join(
                    f"{'[DIR] ' if f.is_dir else ''}{f.name} ({f.size} bytes)" for f in files
                )
                result_content = [{"type": "text", "text": file_list or "Empty directory"}]

            elif tool_name == "e2b_desktop_screenshot":
                result = self._gateway.take_screenshot(e2b_sandbox_id)
                result_content = [
                    {"type": "image", "data": result.image, "mimeType": f"image/{result.format}"}
                ]

            elif tool_name == "e2b_desktop_click":
                x = arguments.get("x", 0)
                y = arguments.get("y", 0)
                button = arguments.get("button", "left")
                result = self._gateway.mouse_click(e2b_sandbox_id, x, y, button)
                result_content = [{"type": "text", "text": f"Clicked at ({x}, {y})"}]

            elif tool_name == "e2b_desktop_type":
                text = arguments.get("text", "")
                result = self._gateway.type_text(e2b_sandbox_id, text)
                result_content = [{"type": "text", "text": f"Typed: {text}"}]

            elif tool_name == "e2b_desktop_code_execute":
                conversation_id = arguments.get("conversation_id")
                code = arguments.get("code", "")
                language = arguments.get("language", "python")
                timeout = arguments.get("timeout", 60)
                if not conversation_id:
                    return self._error_response(request_id, -32602, "conversation_id required")
                # Workspace path for code execution
                workspace_path = f"/home/user/workspace/{conversation_id}"
                result = self._gateway.execute_code(
                    e2b_sandbox_id, code, language, timeout, workspace_path=workspace_path
                )
                output = {
                    "status": result.status,
                    "success": result.is_success(),
                    "stdout": result.stdout,
                    "stderr": result.stderr,
                    "exit_code": result.exit_code,
                    "duration_ms": result.duration_ms,
                }
                if result.error:
                    output["error"] = result.error
                result_content = [{"type": "text", "text": json.dumps(output)}]

            elif tool_name == "e2b_desktop_packages":
                packages = arguments.get("packages", [])
                package_manager = arguments.get("package_manager", "pip")
                result = self._gateway.install_packages(
                    e2b_sandbox_id, packages, package_manager=package_manager
                )
                output = {
                    "status": result.status,
                    "success": result.is_success(),
                    "package_manager": result.package_manager,
                    "packages_installed": result.packages,
                    "output": result.output,
                    "exit_code": result.exit_code,
                    "duration_ms": result.duration_ms,
                }
                if result.error:
                    output["error"] = result.error
                result_content = [{"type": "text", "text": json.dumps(output)}]

            else:
                return self._error_response(request_id, -32601, f"Unknown static tool: {tool_name}")

            return MCPResponse(
                id=request_id,
                result=MCPToolCallResult(content=result_content).model_dump(),
            )

        except Exception as e:
            logger.error(f"Static tool execution error: {e}")
            return self._error_response(request_id, -32000, str(e))

    async def _get_dynamic_tools(self, e2b_sandbox_id: str) -> list[MCPTool]:
        """Get dynamic tools from sandbox's mcp-proxy.

        Dynamic tools are returned without e2b_ prefix (only static tools have prefix).
        """
        # Check cache first
        cached = self._session_store.get_cached_tools(e2b_sandbox_id)
        if cached is not None:
            return [
                MCPTool(
                    name=t.name,  # No prefix for dynamic tools
                    description=t.description,
                    inputSchema=MCPToolInput(**t.input_schema),
                )
                for t in cached
            ]

        # Fetch from sandbox's mcp-proxy
        try:
            mcp_result = self._gateway.call_mcp(e2b_sandbox_id, "tools/list")

            if mcp_result.error:
                logger.warning(f"Failed to get dynamic tools: {mcp_result.error}")
                return []

            if mcp_result.result is None:
                return []

            tools_data = mcp_result.result.get("tools", [])
            tool_infos: list[ToolInfo] = []
            mcp_tools: list[MCPTool] = []

            for tool_data in tools_data:
                tool_name = tool_data.get("name", "")
                tool_desc = tool_data.get("description", "")
                input_schema = tool_data.get("inputSchema", {})

                tool_infos.append(
                    ToolInfo(name=tool_name, description=tool_desc, input_schema=input_schema)
                )
                mcp_tools.append(
                    MCPTool(
                        name=tool_name,  # No prefix for dynamic tools
                        description=tool_desc,
                        inputSchema=MCPToolInput(**input_schema) if input_schema else MCPToolInput(),
                    )
                )

            self._session_store.set_cached_tools(e2b_sandbox_id, tool_infos)
            return mcp_tools

        except Exception as e:
            logger.warning(f"Failed to fetch dynamic tools: {e}")
            return []

    async def _proxy_dynamic_tool_call(
        self, request: MCPRequest, e2b_sandbox_id: str
    ) -> MCPResponse:
        """Proxy a dynamic tool call to sandbox's mcp-proxy.

        Dynamic tools don't have e2b_ prefix, so tool_name is passed as-is.
        """
        try:
            params = request.params or {}
            tool_name = params.get("name", "")
            arguments = params.get("arguments", {})

            mcp_result = self._gateway.call_mcp(
                e2b_sandbox_id, "tools/call", {"name": tool_name, "arguments": arguments}
            )

            if mcp_result.error:
                return self._error_response(
                    request.id,
                    mcp_result.error.get("code", -32000),
                    mcp_result.error.get("message", "Dynamic tool call failed"),
                )

            return MCPResponse(id=request.id, result=mcp_result.result)

        except Exception as e:
            logger.error(f"Dynamic tool proxy error: {e}")
            return self._error_response(request.id, -32000, str(e))

    def _error_response(
        self, request_id: int | str | None, code: int, message: str
    ) -> MCPResponse:
        """Create an error response."""
        return MCPResponse(id=request_id, error=MCPError(code=code, message=message))
