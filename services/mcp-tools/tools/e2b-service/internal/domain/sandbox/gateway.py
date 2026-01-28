"""Sandbox gateway interface (abstract)."""

from abc import ABC, abstractmethod
from typing import Any

from .entity import (
    Sandbox,
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


class SandboxGateway(ABC):
    """Abstract interface for sandbox operations.

    This interface defines the contract for sandbox management.
    Implementations (e.g., E2BGateway) must implement all methods.
    """

    @abstractmethod
    def create_sandbox(self, timeout: int | None = None) -> Sandbox:
        """Create and start a new sandbox.

        Args:
            timeout: Sandbox timeout in seconds

        Returns:
            Sandbox entity with sandbox_id and status
        """
        pass

    @abstractmethod
    def stop_sandbox(self, sandbox_id: str) -> Sandbox:
        """Stop and cleanup a sandbox.

        Args:
            sandbox_id: The sandbox ID to stop

        Returns:
            Sandbox entity with stopped status
        """
        pass

    @abstractmethod
    def pause_sandbox(self, sandbox_id: str) -> bool:
        """Pause a sandbox (preserves state for later resume).

        Args:
            sandbox_id: The sandbox ID to pause

        Returns:
            True if successful, False otherwise
        """
        pass

    @abstractmethod
    def resume_sandbox(self, sandbox_id: str, timeout: int) -> Sandbox | None:
        """Resume a paused sandbox with specified timeout.

        Args:
            sandbox_id: The sandbox ID to resume
            timeout: Timeout in seconds for the resumed sandbox

        Returns:
            Sandbox with updated info, or None if failed
        """
        pass

    @abstractmethod
    def run_command(
        self, sandbox_id: str, command: str, timeout: int = 60, cwd: str | None = None
    ) -> CommandResult:
        """Execute a shell command in the sandbox.

        Args:
            sandbox_id: The sandbox ID
            command: Shell command to execute
            timeout: Command timeout in seconds
            cwd: Optional working directory to run command in

        Returns:
            CommandResult with stdout, stderr, exit_code
        """
        pass

    @abstractmethod
    def read_file(self, sandbox_id: str, path: str) -> FileContent:
        """Read a file from the sandbox filesystem.

        Args:
            sandbox_id: The sandbox ID
            path: Absolute path to the file

        Returns:
            FileContent with base64 encoded content
        """
        pass

    @abstractmethod
    def write_file(self, sandbox_id: str, path: str, content: str) -> FileWriteResult:
        """Write content to a file in the sandbox filesystem.

        Args:
            sandbox_id: The sandbox ID
            path: Absolute path to the file
            content: Content to write

        Returns:
            FileWriteResult with status and bytes written
        """
        pass

    @abstractmethod
    def list_files(self, sandbox_id: str, path: str) -> list[FileInfo]:
        """List files in a directory in the sandbox.

        Args:
            sandbox_id: The sandbox ID
            path: Directory path to list

        Returns:
            List of FileInfo objects
        """
        pass

    @abstractmethod
    def take_screenshot(self, sandbox_id: str) -> Screenshot:
        """Take a screenshot of the sandbox desktop.

        Args:
            sandbox_id: The sandbox ID

        Returns:
            Screenshot with base64 encoded image
        """
        pass

    @abstractmethod
    def open_url(self, sandbox_id: str, url: str) -> BrowserResult:
        """Open a URL in the sandbox browser.

        Args:
            sandbox_id: The sandbox ID
            url: URL to open

        Returns:
            BrowserResult with status
        """
        pass

    @abstractmethod
    def mouse_click(
        self, sandbox_id: str, x: int, y: int, button: str = "left"
    ) -> InputResult:
        """Perform a mouse click at coordinates.

        Args:
            sandbox_id: The sandbox ID
            x: X coordinate
            y: Y coordinate
            button: Mouse button (left, right, middle)

        Returns:
            InputResult with status
        """
        pass

    @abstractmethod
    def type_text(self, sandbox_id: str, text: str) -> InputResult:
        """Type text in the sandbox.

        Args:
            sandbox_id: The sandbox ID
            text: Text to type

        Returns:
            InputResult with status
        """
        pass

    @abstractmethod
    def call_mcp(
        self, sandbox_id: str, method: str, params: dict[str, Any] | None = None
    ) -> MCPResult:
        """Call an MCP method on the sandbox's MCP server.

        Args:
            sandbox_id: The sandbox ID
            method: MCP method name
            params: MCP method parameters

        Returns:
            MCPResult with response or error
        """
        pass

    @abstractmethod
    def is_sandbox_alive(self, sandbox_id: str) -> bool:
        """Check if sandbox is actually alive on E2B platform.

        Args:
            sandbox_id: The E2B sandbox ID

        Returns:
            True if sandbox is reachable, False otherwise
        """
        pass

    @abstractmethod
    def execute_code(
        self,
        sandbox_id: str,
        code: str,
        language: str = "python",
        timeout: int = 60,
        workspace_path: str | None = None,
    ) -> CodeExecuteResult:
        """Execute code in the sandbox.

        Writes code to a temp file and executes via shell.
        Supports Python and Node.js.

        Args:
            sandbox_id: The sandbox ID
            code: Code to execute
            language: Programming language ("python" or "nodejs")
            timeout: Execution timeout in seconds
            workspace_path: If provided, code runs in this directory (cwd)

        Returns:
            CodeExecuteResult with stdout, stderr, exit_code
        """
        pass

    @abstractmethod
    def set_timeout(self, sandbox_id: str, timeout: int) -> bool:
        """Set the timeout for a sandbox.

        Note: set_timeout(X) sets timeout to X seconds FROM NOW, not extends by X.

        Args:
            sandbox_id: The E2B sandbox ID
            timeout: New timeout in seconds from now

        Returns:
            True if successful, False otherwise
        """
        pass

    @abstractmethod
    def get_sandbox_info(self, sandbox_id: str) -> Sandbox | None:
        """Get current sandbox info including timestamps.

        Args:
            sandbox_id: The E2B sandbox ID

        Returns:
            Sandbox with started_at and end_at, or None if not found
        """
        pass

    @abstractmethod
    def install_packages(
        self,
        sandbox_id: str,
        packages: list[str],
        package_manager: str = "pip",
        timeout: int = 120,
    ) -> PackageInstallResult:
        """Install packages using pip or npm.

        Args:
            sandbox_id: The sandbox ID
            packages: List of package names to install
            package_manager: "pip" for Python or "npm" for Node.js
            timeout: Installation timeout in seconds

        Returns:
            PackageInstallResult with status and output
        """
        pass
