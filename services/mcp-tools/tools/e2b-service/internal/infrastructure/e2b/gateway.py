"""E2B SDK Gateway - Implementation of SandboxGateway interface."""

import base64
import json
import logging
import os
import time
import uuid
from typing import Any

import httpx
from e2b_desktop import Sandbox as E2BSandbox

from internal.config import settings
from internal.domain.sandbox import (
    SandboxGateway,
    Sandbox,
    SandboxStatus,
    CommandResult,
    FileInfo,
    FileContent,
    FileWriteResult,
    Screenshot,
    BrowserResult,
    InputResult,
    MCPResult,
    CodeExecuteResult,
    PackageInstallResult,
)

logger = logging.getLogger(__name__)


class E2BGateway(SandboxGateway):
    """E2B implementation of the SandboxGateway interface."""

    # MCP proxy HTTP port (mcp-proxy exposes HTTP endpoint on this port)
    MCP_SERVER_PORT = 17390

    def __init__(self, api_key: str, template_id: str):
        """Initialize gateway with injected configuration.

        Args:
            api_key: E2B API key
            template_id: E2B sandbox template ID
        """
        self.api_key = api_key
        self.template_id = template_id
        # E2B SDK reads from environment variable
        os.environ["E2B_API_KEY"] = api_key
        self._sandboxes: dict[str, E2BSandbox] = {}
        # MCP session state per sandbox
        self._mcp_sessions: dict[str, dict] = {}  # sandbox_id -> {session_id, initialized}

    def _connect(self, sandbox_id: str) -> E2BSandbox:
        """Connect to an existing sandbox."""
        return E2BSandbox.connect(sandbox_id)

    def create_sandbox(self, timeout: int | None = None) -> Sandbox:
        """Create and start a new E2B sandbox."""
        timeout = timeout or settings.e2b_sandbox_timeout

        logger.info(f"Creating sandbox with template={self.template_id}, timeout={timeout}s")

        # Use beta_create to enable auto_pause - sandbox pauses on timeout instead of being killed
        # This allows resuming the sandbox later
        sandbox = E2BSandbox.beta_create(
            template=self.template_id,
            timeout=timeout,
            auto_pause=True,
        )

        sandbox_id = sandbox.sandbox_id
        self._sandboxes[sandbox_id] = sandbox

        logger.info(f"Sandbox created: {sandbox_id}")

        # Launch Chromium with extension using startup script
        sandbox.commands.run(
            "/usr/local/bin/start-browser.sh https://google.com",
            background=True,
            timeout=0,
        )
        logger.info("Chromium started with extension")

        # Start VNC streaming with token auth for security
        sandbox.stream.start(require_auth=True)

        # Get stream URL with auth token (control URL for interaction)
        view_url = None
        control_url = None
        sandbox_url = None
        try:
            # Get auth key for secure URL access
            auth_key = sandbox.stream.get_auth_key()
            control_url = sandbox.stream.get_url(auth_key=auth_key)
            # View-only URL with same auth key
            view_url = sandbox.stream.get_url(auth_key=auth_key, view_only=True)

            # Build sandbox URL for external MCP access (via E2B's port forwarding)
            mcp_host = sandbox.get_host(self.MCP_SERVER_PORT)
            sandbox_url = f"https://{mcp_host}"
            logger.info(f"Sandbox URL: {sandbox_url}")
        except Exception as e:
            logger.warning(f"Failed to get stream/sandbox URLs for {sandbox_id}: {e}")

        # Reset timeout AFTER initialization completes
        # This gives the user the full requested timeout from when sandbox is ready to use
        # (initialization can take 30-40 seconds which eats into the original timeout)
        try:
            sandbox.set_timeout(timeout)
            logger.info(f"Reset timeout for {sandbox_id} to {timeout}s after initialization")
        except Exception as e:
            logger.warning(f"Failed to reset timeout for {sandbox_id}: {e}")

        # Get sandbox info for proper timestamps (after reset)
        started_at = None
        end_at = None
        try:
            info = sandbox.get_info()
            started_at = info.started_at
            end_at = info.end_at
            logger.info(f"Sandbox {sandbox_id} started_at={started_at}, end_at={end_at}")
        except Exception as e:
            logger.warning(f"Failed to get sandbox info for {sandbox_id}: {e}")

        return Sandbox(
            sandbox_id=sandbox_id,
            status=SandboxStatus.RUNNING,
            view_url=view_url,
            control_url=control_url,
            sandbox_url=sandbox_url,
            started_at=started_at,
            end_at=end_at,
        )

    def _get_or_connect_sandbox(self, sandbox_id: str) -> E2BSandbox | None:
        """Get sandbox from cache or try to connect to existing one."""
        sandbox = self._sandboxes.get(sandbox_id)
        if sandbox:
            return sandbox
        try:
            sandbox = self._connect(sandbox_id)
            self._sandboxes[sandbox_id] = sandbox
            return sandbox
        except Exception as e:
            logger.debug(f"Failed to connect to sandbox {sandbox_id}: {e}")
            return None

    def stop_sandbox(self, sandbox_id: str) -> Sandbox:
        """Stop and cleanup a sandbox."""
        sandbox = self._sandboxes.pop(sandbox_id, None)
        if not sandbox:
            # Try to connect to existing sandbox
            try:
                sandbox = self._connect(sandbox_id)
            except Exception:
                return Sandbox(
                    sandbox_id=sandbox_id,
                    status=SandboxStatus.NOT_FOUND,
                )

        try:
            sandbox.kill()
            logger.info(f"Sandbox stopped: {sandbox_id}")
            return Sandbox(
                sandbox_id=sandbox_id,
                status=SandboxStatus.STOPPED,
            )
        except Exception as e:
            logger.error(f"Error stopping sandbox {sandbox_id}: {e}")
            return Sandbox(
                sandbox_id=sandbox_id,
                status=SandboxStatus.ERROR,
            )

    def pause_sandbox(self, sandbox_id: str) -> bool:
        """Pause a sandbox using E2B beta_pause."""
        try:
            sandbox = self._get_or_connect_sandbox(sandbox_id)
            if not sandbox:
                logger.error(f"Cannot pause: sandbox {sandbox_id} not found")
                return False

            sandbox.beta_pause()
            # Remove from cache since it's now paused
            self._sandboxes.pop(sandbox_id, None)
            logger.info(f"Paused sandbox {sandbox_id}")
            return True
        except Exception as e:
            logger.error(f"Failed to pause sandbox {sandbox_id}: {e}")
            return False

    def resume_sandbox(self, sandbox_id: str, timeout: int) -> Sandbox | None:
        """Resume a paused sandbox with specified timeout."""
        try:
            # Connect to the paused sandbox - this resumes it
            # Note: E2B connect uses default 5min timeout, so we set it after
            sandbox = E2BSandbox.connect(sandbox_id)
            self._sandboxes[sandbox_id] = sandbox

            # Set the proper timeout (E2B connect defaults to 5 min)
            sandbox.set_timeout(timeout)
            logger.info(f"Resumed sandbox {sandbox_id} with timeout={timeout}s")

            # Get updated info
            info = sandbox.get_info()
            return Sandbox(
                sandbox_id=sandbox_id,
                status=SandboxStatus.RUNNING,
                started_at=info.started_at,
                end_at=info.end_at,
            )
        except Exception as e:
            logger.error(f"Failed to resume sandbox {sandbox_id}: {e}")
            return None

    def run_command(
        self, sandbox_id: str, command: str, timeout: int = 60, cwd: str | None = None
    ) -> CommandResult:
        """Execute a shell command in the sandbox.

        Args:
            sandbox_id: The E2B sandbox ID
            command: Shell command to execute
            timeout: Timeout in seconds (default: 60)
            cwd: Optional working directory to run command in
        """
        sandbox = self._get_or_connect_sandbox(sandbox_id)
        if not sandbox:
            return CommandResult(error=f"Sandbox {sandbox_id} not found")

        try:
            # If cwd specified, wrap command with cd
            if cwd:
                command = f"cd {cwd} && {command}"

            result = sandbox.commands.run(command, timeout=timeout)
            return CommandResult(
                stdout=result.stdout,
                stderr=result.stderr,
                exit_code=result.exit_code,
            )
        except Exception as e:
            logger.error(f"Command execution error in {sandbox_id}: {e}")
            return CommandResult(error=str(e))

    def read_file(self, sandbox_id: str, path: str) -> FileContent:
        """Read a file from the sandbox filesystem."""
        sandbox = self._get_or_connect_sandbox(sandbox_id)
        if not sandbox:
            raise ValueError(f"Sandbox {sandbox_id} not found")

        try:
            content = sandbox.files.read(path)
            # Encode as base64 for safe transport
            if isinstance(content, bytes):
                encoded = base64.b64encode(content).decode("utf-8")
            else:
                encoded = base64.b64encode(content.encode("utf-8")).decode("utf-8")
            return FileContent(path=path, content=encoded, encoding="base64")
        except Exception as e:
            logger.error(f"File read error in {sandbox_id}: {e}")
            raise

    def write_file(self, sandbox_id: str, path: str, content: str) -> FileWriteResult:
        """Write content to a file in the sandbox filesystem."""
        sandbox = self._get_or_connect_sandbox(sandbox_id)
        if not sandbox:
            raise ValueError(f"Sandbox {sandbox_id} not found")

        try:
            sandbox.files.write(path, content)
            return FileWriteResult(
                path=path,
                status="written",
                bytes_written=len(content),
            )
        except Exception as e:
            logger.error(f"File write error in {sandbox_id}: {e}")
            raise

    def list_files(self, sandbox_id: str, path: str) -> list[FileInfo]:
        """List files in a directory in the sandbox."""
        sandbox = self._get_or_connect_sandbox(sandbox_id)
        if not sandbox:
            raise ValueError(f"Sandbox {sandbox_id} not found")

        try:
            files = sandbox.files.list(path)
            result = []
            for f in files:
                # E2B SDK may expose a FileType enum or a plain string.
                file_type = getattr(f, "type", None)
                if isinstance(file_type, str):
                    is_dir = file_type == "dir"
                else:
                    is_dir = file_type is not None and getattr(file_type, "value", "") == "dir"
                result.append(
                    FileInfo(
                        name=f.name,
                        is_dir=is_dir,
                        size=getattr(f, "size", 0),
                    )
                )
            return result
        except Exception as e:
            logger.error(f"File list error in {sandbox_id}: {e}")
            raise

    def take_screenshot(self, sandbox_id: str) -> Screenshot:
        """Take a screenshot of the sandbox desktop."""
        sandbox = self._get_or_connect_sandbox(sandbox_id)
        if not sandbox:
            raise ValueError(f"Sandbox {sandbox_id} not found")

        try:
            screenshot = sandbox.screenshot()
            encoded = base64.b64encode(screenshot).decode("utf-8")
            return Screenshot(image=encoded, format="png", encoding="base64")
        except Exception as e:
            logger.error(f"Screenshot error in {sandbox_id}: {e}")
            raise

    def open_url(self, sandbox_id: str, url: str) -> BrowserResult:
        """Open a URL in the sandbox browser."""
        sandbox = self._get_or_connect_sandbox(sandbox_id)
        if not sandbox:
            raise ValueError(f"Sandbox {sandbox_id} not found")

        try:
            sandbox.open(url)
            return BrowserResult(url=url, status="opened")
        except Exception as e:
            logger.error(f"Open URL error in {sandbox_id}: {e}")
            raise

    def mouse_click(
        self, sandbox_id: str, x: int, y: int, button: str = "left"
    ) -> InputResult:
        """Perform a mouse click at coordinates."""
        sandbox = self._get_or_connect_sandbox(sandbox_id)
        if not sandbox:
            raise ValueError(f"Sandbox {sandbox_id} not found")

        try:
            btn = (button or "left").lower()
            if btn == "right":
                sandbox.right_click(x=x, y=y)
            elif btn == "middle":
                sandbox.middle_click(x=x, y=y)
            else:
                sandbox.left_click(x=x, y=y)
            return InputResult(status="clicked", x=x, y=y, button=btn)
        except Exception as e:
            logger.error(f"Mouse click error in {sandbox_id}: {e}")
            raise

    def type_text(self, sandbox_id: str, text: str) -> InputResult:
        """Type text in the sandbox."""
        sandbox = self._get_or_connect_sandbox(sandbox_id)
        if not sandbox:
            raise ValueError(f"Sandbox {sandbox_id} not found")

        try:
            sandbox.write(text)
            return InputResult(status="typed", text=text)
        except Exception as e:
            logger.error(f"Type text error in {sandbox_id}: {e}")
            raise

    def _get_sandbox_url(self, sandbox_id: str) -> str | None:
        """Get external sandbox URL for MCP proxy access."""
        sandbox = self._get_or_connect_sandbox(sandbox_id)
        if not sandbox:
            return None
        try:
            mcp_host = sandbox.get_host(self.MCP_SERVER_PORT)
            return f"https://{mcp_host}/mcp"
        except Exception as e:
            logger.error(f"Failed to get sandbox URL for {sandbox_id}: {e}")
            return None

    def _parse_sse_response(self, text: str) -> dict:
        """Parse SSE response to extract JSON-RPC message."""
        for line in text.split("\n"):
            if line.startswith("data: "):
                return json.loads(line[6:])
        raise ValueError(f"No data line found in SSE response: {text}")

    def _ensure_mcp_initialized(self, sandbox_id: str, mcp_url: str) -> str | None:
        """Ensure MCP session is initialized. Returns session ID."""
        session = self._mcp_sessions.get(sandbox_id, {})
        if session.get("initialized"):
            return session.get("session_id")

        # Initialize session
        try:
            headers = {
                "Content-Type": "application/json",
                "Accept": "application/json, text/event-stream",
            }
            request_body = {
                "jsonrpc": "2.0",
                "method": "initialize",
                "params": {
                    "protocolVersion": "2024-11-05",
                    "capabilities": {},
                    "clientInfo": {"name": "e2b-api", "version": "1.0"},
                },
                "id": 0,
            }

            with httpx.Client(timeout=30) as client:
                response = client.post(mcp_url, json=request_body, headers=headers)
                response.raise_for_status()

                # Capture session ID from response header
                session_id = response.headers.get("mcp-session-id")

                self._mcp_sessions[sandbox_id] = {
                    "session_id": session_id,
                    "initialized": True,
                }
                logger.info(f"MCP session initialized for {sandbox_id}, session_id={session_id}")
                return session_id

        except Exception as e:
            logger.error(f"Failed to initialize MCP session for {sandbox_id}: {e}")
            return None

    def call_mcp(
        self, sandbox_id: str, method: str, params: dict[str, Any] | None = None
    ) -> MCPResult:
        """Call an MCP method on the sandbox's MCP server via external URL."""
        mcp_url = self._get_sandbox_url(sandbox_id)
        if not mcp_url:
            return MCPResult(error={"code": -32000, "message": f"Sandbox {sandbox_id} not found"})

        try:
            # Ensure session is initialized
            session_id = self._ensure_mcp_initialized(sandbox_id, mcp_url)

            # Build headers
            headers = {
                "Content-Type": "application/json",
                "Accept": "application/json, text/event-stream",
            }
            if session_id:
                headers["mcp-session-id"] = session_id

            # Build JSON-RPC request
            request_body = {
                "jsonrpc": "2.0",
                "method": method,
                "params": params or {},
                "id": 1,
            }

            # Make external HTTP request
            with httpx.Client(timeout=60) as client:
                response = client.post(mcp_url, json=request_body, headers=headers)
                response.raise_for_status()

                # Update session ID if returned
                if "mcp-session-id" in response.headers:
                    self._mcp_sessions[sandbox_id]["session_id"] = response.headers["mcp-session-id"]

                # Parse response (may be SSE or JSON)
                content_type = response.headers.get("content-type", "")
                if "text/event-stream" in content_type:
                    result = self._parse_sse_response(response.text)
                else:
                    result = response.json()

                return MCPResult(
                    result=result.get("result"),
                    error=result.get("error"),
                )

        except Exception as e:
            logger.error(f"MCP call error for {sandbox_id}: {e}")
            return MCPResult(error={"code": -32603, "message": str(e)})

    def is_sandbox_alive(self, sandbox_id: str) -> bool:
        """Check if sandbox is actually alive by attempting to connect."""
        try:
            sandbox = self._get_or_connect_sandbox(sandbox_id)
            if not sandbox:
                return False
            # Quick ping to verify it's actually responsive
            result = sandbox.commands.run("echo ping", timeout=5)
            return result.exit_code == 0
        except Exception as e:
            logger.debug(f"Sandbox {sandbox_id} is not alive: {e}")
            # Any error means sandbox is not alive
            return False

    def set_timeout(self, sandbox_id: str, timeout: int) -> bool:
        """Set the timeout for a sandbox on E2B platform.

        Note: set_timeout(X) sets timeout to X seconds FROM NOW.
        """
        try:
            sandbox = self._get_or_connect_sandbox(sandbox_id)
            if not sandbox:
                logger.error(f"Cannot set timeout: sandbox {sandbox_id} not found")
                return False
            sandbox.set_timeout(timeout)
            logger.info(f"Set timeout for sandbox {sandbox_id} to {timeout}s from now")
            return True
        except Exception as e:
            logger.error(f"Failed to set timeout for sandbox {sandbox_id}: {e}")
            return False

    def get_sandbox_info(self, sandbox_id: str) -> Sandbox | None:
        """Get current sandbox info including timestamps from E2B."""
        try:
            sandbox = self._get_or_connect_sandbox(sandbox_id)
            if not sandbox:
                return None
            info = sandbox.get_info()
            return Sandbox(
                sandbox_id=sandbox_id,
                status=SandboxStatus.RUNNING,
                started_at=info.started_at,
                end_at=info.end_at,
            )
        except Exception as e:
            logger.error(f"Failed to get sandbox info for {sandbox_id}: {e}")
            return None

    def execute_code(
        self,
        sandbox_id: str,
        code: str,
        language: str = "python",
        timeout: int = 60,
        workspace_path: str | None = None,
    ) -> CodeExecuteResult:
        """Execute code in the sandbox via temp file.

        Writes code to a temp file and executes via shell command.
        If workspace_path is provided, code runs in that directory.
        """
        start_time = time.time()
        sandbox = self._get_or_connect_sandbox(sandbox_id)
        if not sandbox:
            return CodeExecuteResult(
                status="error",
                error=f"Sandbox {sandbox_id} not found",
                duration_ms=int((time.time() - start_time) * 1000),
            )

        # Validate language
        lang = language.lower()
        if lang not in ("python", "nodejs", "javascript"):
            return CodeExecuteResult(
                status="error",
                error=f"Unsupported language: {language} (use 'python' or 'nodejs')",
                duration_ms=int((time.time() - start_time) * 1000),
            )

        # Normalize language
        if lang in ("nodejs", "javascript"):
            lang = "nodejs"

        # Generate temp file path (in workspace or /tmp)
        exec_id = uuid.uuid4().hex[:12]
        if workspace_path:
            # Code runs in workspace directory
            base_dir = workspace_path.rstrip("/")
            if lang == "python":
                temp_file = f"{base_dir}/.exec_{exec_id}.py"
                run_cmd = f"cd {base_dir} && python3 {temp_file}"
            else:  # nodejs
                temp_file = f"{base_dir}/.exec_{exec_id}.js"
                run_cmd = f"cd {base_dir} && node {temp_file}"
        else:
            # Default: run in /tmp
            if lang == "python":
                temp_file = f"/tmp/exec_{exec_id}.py"
                run_cmd = f"python3 {temp_file}"
            else:  # nodejs
                temp_file = f"/tmp/exec_{exec_id}.js"
                run_cmd = f"node {temp_file}"

        try:
            # Write code to temp file
            sandbox.files.write(temp_file, code)
            logger.debug(f"Code written to {temp_file} ({len(code)} bytes)")

            # Execute the code
            result = sandbox.commands.run(run_cmd, timeout=timeout)

            # Cleanup temp file (best effort)
            try:
                sandbox.commands.run(f"rm -f {temp_file}", timeout=5)
            except Exception:
                pass  # Ignore cleanup errors

            duration_ms = int((time.time() - start_time) * 1000)
            exit_code = result.exit_code or 0
            status = "ok" if exit_code == 0 else "error"

            return CodeExecuteResult(
                stdout=result.stdout or "",
                stderr=result.stderr or "",
                exit_code=exit_code,
                status=status,
                error=None if status == "ok" else f"Exit code: {exit_code}",
                duration_ms=duration_ms,
            )

        except Exception as e:
            logger.error(f"Code execution error in {sandbox_id}: {e}")
            # Try to cleanup
            try:
                sandbox.commands.run(f"rm -f {temp_file}", timeout=5)
            except Exception:
                pass
            return CodeExecuteResult(
                status="error",
                error=str(e),
                duration_ms=int((time.time() - start_time) * 1000),
            )

    def install_packages(
        self,
        sandbox_id: str,
        packages: list[str],
        package_manager: str = "pip",
        timeout: int = 120,
    ) -> PackageInstallResult:
        """Install packages using pip or npm."""
        start_time = time.time()
        sandbox = self._get_or_connect_sandbox(sandbox_id)
        if not sandbox:
            return PackageInstallResult(
                packages=packages,
                package_manager=package_manager,
                status="error",
                error=f"Sandbox {sandbox_id} not found",
                duration_ms=int((time.time() - start_time) * 1000),
            )

        if not packages:
            return PackageInstallResult(
                packages=[],
                package_manager=package_manager,
                status="error",
                error="No packages specified",
                duration_ms=int((time.time() - start_time) * 1000),
            )

        # Validate package manager
        pm = package_manager.lower()
        if pm not in ("pip", "npm"):
            return PackageInstallResult(
                packages=packages,
                package_manager=package_manager,
                status="error",
                error=f"Unsupported package manager: {package_manager} (use 'pip' or 'npm')",
                duration_ms=int((time.time() - start_time) * 1000),
            )

        # Build install command
        packages_str = " ".join(packages)
        if pm == "pip":
            command = f"pip install --quiet --disable-pip-version-check {packages_str}"
        else:  # npm
            command = f"npm install -g {packages_str}"

        logger.info(f"Installing packages via {pm} in {sandbox_id}: {packages}")

        try:
            result = sandbox.commands.run(command, timeout=timeout)

            duration_ms = int((time.time() - start_time) * 1000)
            exit_code = result.exit_code or 0
            output = (result.stdout or "") + (result.stderr or "")

            if exit_code == 0:
                logger.info(f"Packages installed successfully via {pm}: {packages}")
                return PackageInstallResult(
                    packages=packages,
                    package_manager=pm,
                    status="success",
                    output=output.strip(),
                    exit_code=exit_code,
                    duration_ms=duration_ms,
                )
            else:
                logger.error(f"{pm} install failed with exit code {exit_code}")
                return PackageInstallResult(
                    packages=packages,
                    package_manager=pm,
                    status="error",
                    output=output.strip(),
                    exit_code=exit_code,
                    error=f"{pm} install failed with exit code {exit_code}",
                    duration_ms=duration_ms,
                )

        except Exception as e:
            logger.error(f"Package installation error in {sandbox_id}: {e}")
            return PackageInstallResult(
                packages=packages,
                package_manager=pm,
                status="error",
                error=str(e),
                duration_ms=int((time.time() - start_time) * 1000),
            )
