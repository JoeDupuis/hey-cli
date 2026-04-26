package cmd

import (
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/basecamp/hey-cli/internal/output"
)

// registerMCPTools registers every hey-cli MCP tool on the given server.
// All handlers reuse the package-level `sdk` client and helpers from sdk.go.
func registerMCPTools(server *mcp.Server) {
	registerMailTools(server)
	registerCalendarTools(server)
}

// toolError converts SDK / hey-cli errors into a structured error string suitable
// for an MCP tool error response (CallToolResult.IsError = true). Returning an
// error from a ToolHandlerFor is treated as a tool error rather than a protocol
// error by the SDK.
//
// If err is already an *output.Error (e.g. constructed by output.ErrNotFound),
// it is formatted as-is — convertSDKError only understands SDK errors and would
// otherwise drop the code and message into a blank ErrAPI default branch.
func toolError(err error) error {
	if err == nil {
		return nil
	}

	var oe *output.Error
	if !errors.As(err, &oe) {
		mapped := convertSDKError(err)
		if !errors.As(mapped, &oe) {
			return mapped
		}
	}

	if oe.HTTPStatus > 0 {
		return fmt.Errorf("%s: %s (http %d)", oe.Code, oe.Message, oe.HTTPStatus)
	}
	return fmt.Errorf("%s: %s", oe.Code, oe.Message)
}
