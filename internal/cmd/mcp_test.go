package cmd

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/basecamp/hey-cli/internal/output"
)

// TestRegisterMCPTools verifies registration completes without panicking.
// AddTool panics on malformed Tool definitions (bad name characters, schemas
// it can't infer from In/Out, etc.), so a successful build of the server is a
// real signal that every handler signature and Tool struct in the catalog is
// well-formed.
func TestRegisterMCPTools(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("registerMCPTools panicked: %v", r)
		}
	}()

	server := mcp.NewServer(&mcp.Implementation{Name: "hey-test", Version: "0"}, nil)
	registerMCPTools(server)
}

// TestToolErrorPreservesOutputError ensures errors that are already *output.Error
// (e.g. output.ErrNotFound) round-trip with their code and message intact, rather
// than being flattened into a generic API error by convertSDKError.
func TestToolErrorPreservesOutputError(t *testing.T) {
	in := output.ErrNotFound("thing", "42")
	got := toolError(in)
	if got == nil {
		t.Fatal("toolError returned nil for non-nil input")
	}
	msg := got.Error()
	if !strings.Contains(msg, "not_found") {
		t.Errorf("expected error %q to contain code %q", msg, "not_found")
	}
	if !strings.Contains(msg, "42") {
		t.Errorf("expected error %q to contain identifier %q", msg, "42")
	}
}

func TestToolErrorNilPassthrough(t *testing.T) {
	if got := toolError(nil); got != nil {
		t.Errorf("toolError(nil) = %v, want nil", got)
	}
}
