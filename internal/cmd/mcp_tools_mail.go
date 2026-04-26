package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/basecamp/hey-sdk/go/pkg/generated"

	"github.com/basecamp/hey-cli/internal/htmlutil"
	"github.com/basecamp/hey-cli/internal/output"
)

// --- search ---

type searchTopicsInput struct {
	Q    string `json:"q" jsonschema:"free-text query, e.g. \"invoice\""`
	Page string `json:"page,omitempty" jsonschema:"optional pagination cursor returned by a previous search call"`
}

func searchTopicsHandler(ctx context.Context, _ *mcp.CallToolRequest, in searchTopicsInput) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(in.Q) == "" {
		return nil, nil, fmt.Errorf("q is required")
	}
	result, err := sdk.Search().Search(ctx, &generated.SearchParams{Q: in.Q, Page: in.Page})
	if err != nil {
		return nil, nil, toolError(err)
	}
	return nil, result, nil
}

// --- list_boxes ---

func listBoxesHandler(ctx context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	result, err := sdk.Boxes().List(ctx)
	if err != nil {
		return nil, nil, toolError(err)
	}
	var boxes []generated.Box
	if result != nil {
		boxes = *result
	}
	return nil, boxes, nil
}

// --- list_box_postings ---

type listBoxPostingsInput struct {
	Box   string `json:"box" jsonschema:"box name (imbox, feedbox, trailbox, asidebox, laterbox, bubblebox) or numeric box id"`
	Limit int    `json:"limit,omitempty" jsonschema:"max number of postings to return (0 = first page from server)"`
}

func listBoxPostingsHandler(ctx context.Context, _ *mcp.CallToolRequest, in listBoxPostingsInput) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(in.Box) == "" {
		return nil, nil, fmt.Errorf("box is required")
	}
	resp, err := resolveBox(ctx, in.Box)
	if err != nil {
		return nil, nil, toolError(err)
	}

	postings := resp.Postings
	if in.Limit > 0 && len(postings) > in.Limit {
		postings = postings[:in.Limit]
		resp.NextHistoryUrl = ""
	}
	resp.Postings = postings
	return nil, resp, nil
}

// --- read_thread ---

type readThreadInput struct {
	ThreadID int64 `json:"thread_id" jsonschema:"numeric thread (topic) id"`
}

func readThreadHandler(ctx context.Context, _ *mcp.CallToolRequest, in readThreadInput) (*mcp.CallToolResult, any, error) {
	if in.ThreadID == 0 {
		return nil, nil, fmt.Errorf("thread_id is required")
	}
	resp, err := sdk.GetHTML(ctx, fmt.Sprintf("/topics/%d/entries", in.ThreadID))
	if err != nil {
		return nil, nil, toolError(err)
	}
	entries := htmlutil.ParseTopicEntriesHTML(string(resp.Data))
	return nil, map[string]any{
		"thread_id": in.ThreadID,
		"entries":   entries,
	}, nil
}

// --- list_drafts ---

type listDraftsInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"max number of drafts to return (0 = no limit)"`
}

func listDraftsHandler(ctx context.Context, _ *mcp.CallToolRequest, in listDraftsInput) (*mcp.CallToolResult, any, error) {
	result, err := sdk.Entries().ListDrafts(ctx, nil)
	if err != nil {
		return nil, nil, toolError(err)
	}
	var drafts []generated.DraftMessage
	if result != nil {
		drafts = *result
	}
	if in.Limit > 0 && len(drafts) > in.Limit {
		drafts = drafts[:in.Limit]
	}
	return nil, drafts, nil
}

// --- compose_message ---

type composeMessageInput struct {
	Subject  string   `json:"subject" jsonschema:"message subject (required)"`
	Body     string   `json:"body" jsonschema:"message body in plain text or HTML"`
	To       []string `json:"to,omitempty" jsonschema:"recipient email addresses (required when thread_id is not set)"`
	CC       []string `json:"cc,omitempty" jsonschema:"CC recipient email addresses"`
	BCC      []string `json:"bcc,omitempty" jsonschema:"BCC recipient email addresses"`
	ThreadID int64    `json:"thread_id,omitempty" jsonschema:"if set, post the body into this existing thread instead of starting a new one"`
}

func composeMessageHandler(ctx context.Context, _ *mcp.CallToolRequest, in composeMessageInput) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(in.Subject) == "" {
		return nil, nil, fmt.Errorf("subject is required")
	}
	if strings.TrimSpace(in.Body) == "" {
		return nil, nil, fmt.Errorf("body is required")
	}

	if in.ThreadID != 0 {
		if err := sdk.Messages().CreateTopicMessage(ctx, in.ThreadID, in.Body); err != nil {
			return nil, nil, toolError(err)
		}
		return nil, map[string]any{"sent": true, "thread_id": in.ThreadID}, nil
	}

	if len(in.To) == 0 {
		return nil, nil, fmt.Errorf("to is required when thread_id is not set")
	}
	if err := sdk.Messages().Create(ctx, in.Subject, in.Body, in.To, in.CC, in.BCC); err != nil {
		return nil, nil, toolError(err)
	}
	return nil, map[string]any{"sent": true}, nil
}

// --- reply_to_thread ---

type replyToThreadInput struct {
	ThreadID int64  `json:"thread_id" jsonschema:"numeric thread (topic) id to reply to"`
	Body     string `json:"body" jsonschema:"reply body in plain text or HTML"`
}

func replyToThreadHandler(ctx context.Context, _ *mcp.CallToolRequest, in replyToThreadInput) (*mcp.CallToolResult, any, error) {
	if in.ThreadID == 0 {
		return nil, nil, fmt.Errorf("thread_id is required")
	}
	if strings.TrimSpace(in.Body) == "" {
		return nil, nil, fmt.Errorf("body is required")
	}

	topicResp, err := sdk.GetHTML(ctx, fmt.Sprintf("/topics/%d", in.ThreadID))
	if err != nil {
		return nil, nil, toolError(err)
	}
	addressed := htmlutil.ParseTopicAddressed(string(topicResp.Data))
	if len(addressed.To) == 0 && len(addressed.CC) == 0 && len(addressed.BCC) == 0 {
		return nil, nil, fmt.Errorf("could not determine thread recipients")
	}

	entriesResp, err := sdk.GetHTML(ctx, fmt.Sprintf("/topics/%d/entries", in.ThreadID))
	if err != nil {
		return nil, nil, toolError(err)
	}
	entries := htmlutil.ParseTopicEntriesHTML(string(entriesResp.Data))
	if len(entries) == 0 {
		return nil, nil, toolError(output.ErrNotFound("entries for thread", strconv.FormatInt(in.ThreadID, 10)))
	}
	latestEntryID := entries[len(entries)-1].ID

	if err := sdk.Entries().CreateReply(ctx, latestEntryID, in.Body, addressed.To, addressed.CC, addressed.BCC); err != nil {
		return nil, nil, toolError(err)
	}
	return nil, map[string]any{"sent": true, "thread_id": in.ThreadID, "in_reply_to_entry_id": latestEntryID}, nil
}

// --- mark_seen / mark_unseen ---

type markPostingsInput struct {
	PostingIDs []int64 `json:"posting_ids" jsonschema:"one or more posting ids (NOT thread ids) to update"`
}

func markSeenHandler(ctx context.Context, _ *mcp.CallToolRequest, in markPostingsInput) (*mcp.CallToolResult, any, error) {
	if len(in.PostingIDs) == 0 {
		return nil, nil, fmt.Errorf("posting_ids must contain at least one id")
	}
	if err := sdk.Postings().MarkSeen(ctx, in.PostingIDs); err != nil {
		return nil, nil, toolError(err)
	}
	return nil, map[string]any{"marked": len(in.PostingIDs), "state": "seen"}, nil
}

func markUnseenHandler(ctx context.Context, _ *mcp.CallToolRequest, in markPostingsInput) (*mcp.CallToolResult, any, error) {
	if len(in.PostingIDs) == 0 {
		return nil, nil, fmt.Errorf("posting_ids must contain at least one id")
	}
	if err := sdk.Postings().MarkUnseen(ctx, in.PostingIDs); err != nil {
		return nil, nil, toolError(err)
	}
	return nil, map[string]any{"marked": len(in.PostingIDs), "state": "unseen"}, nil
}

// --- registration ---

func registerMailTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_topics",
		Description: "Full-text search HEY threads (topics). Returns matching topics with metadata. Use the optional page cursor for pagination.",
	}, searchTopicsHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_boxes",
		Description: "List the user's mailboxes (imbox, feedbox, trailbox, asidebox, laterbox, bubblebox, plus any custom boxes).",
	}, listBoxesHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_box_postings",
		Description: "List postings (threads) in a mailbox. Accepts a well-known box name (imbox, feedbox, trailbox, asidebox, laterbox, bubblebox) or a numeric box id.",
	}, listBoxPostingsHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "read_thread",
		Description: "Read a thread: returns every entry (message) in the thread with sender, timestamp, plain-text body, and entry id.",
	}, readThreadHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_drafts",
		Description: "List the user's saved draft messages.",
	}, listDraftsHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "compose_message",
		Description: "Send a new email. Requires subject and body. Provide `to` (and optional `cc`/`bcc`) for a new thread, or `thread_id` to post a new message into an existing thread.",
	}, composeMessageHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "reply_to_thread",
		Description: "Reply to an existing thread. Recipients are inferred from the thread; the latest entry is replied to.",
	}, replyToThreadHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "mark_seen",
		Description: "Mark one or more postings as seen (read). Takes posting ids, not thread ids.",
	}, markSeenHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "mark_unseen",
		Description: "Mark one or more postings as unseen (unread). Takes posting ids, not thread ids.",
	}, markUnseenHandler)
}
