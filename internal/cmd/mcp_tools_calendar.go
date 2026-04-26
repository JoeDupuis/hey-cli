package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/basecamp/hey-sdk/go/pkg/generated"
)

// --- list_calendars ---

func listCalendarsHandler(ctx context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	payload, err := sdk.Calendars().List(ctx)
	if err != nil {
		return nil, nil, toolError(err)
	}
	return nil, unwrapCalendars(payload), nil
}

// --- list_recordings ---

type listRecordingsInput struct {
	CalendarID int64  `json:"calendar_id" jsonschema:"numeric calendar id (use list_calendars to discover)"`
	StartsOn   string `json:"starts_on,omitempty" jsonschema:"start date in YYYY-MM-DD (default: today)"`
	EndsOn     string `json:"ends_on,omitempty" jsonschema:"end date in YYYY-MM-DD (default: starts_on + 30 days)"`
}

func listRecordingsHandler(ctx context.Context, _ *mcp.CallToolRequest, in listRecordingsInput) (*mcp.CallToolResult, any, error) {
	if in.CalendarID == 0 {
		return nil, nil, fmt.Errorf("calendar_id is required")
	}

	startsOn := in.StartsOn
	if startsOn == "" {
		startsOn = time.Now().Format("2006-01-02")
	}
	endsOn := in.EndsOn
	if endsOn == "" {
		start, err := time.Parse("2006-01-02", startsOn)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid starts_on: %s", startsOn)
		}
		endsOn = start.AddDate(0, 0, 30).Format("2006-01-02")
	}

	resp, err := sdk.Calendars().GetRecordings(ctx, in.CalendarID, &generated.GetCalendarRecordingsParams{
		StartsOn: startsOn,
		EndsOn:   endsOn,
	})
	if err != nil {
		return nil, nil, toolError(err)
	}
	return nil, resp, nil
}

// --- todos ---

type listTodosInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"max number of todos to return (0 = no limit)"`
}

func listTodosHandler(ctx context.Context, _ *mcp.CallToolRequest, in listTodosInput) (*mcp.CallToolResult, any, error) {
	resp, err := listPersonalRecordings(ctx)
	if err != nil {
		return nil, nil, toolError(err)
	}
	todos := filterRecordingsByType(resp, "Calendar::Todo")
	if in.Limit > 0 && len(todos) > in.Limit {
		todos = todos[:in.Limit]
	}
	return nil, todos, nil
}

type createTodoInput struct {
	Title string `json:"title" jsonschema:"todo title"`
	Date  string `json:"date,omitempty" jsonschema:"due date in YYYY-MM-DD (optional)"`
}

func createTodoHandler(ctx context.Context, _ *mcp.CallToolRequest, in createTodoInput) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, nil, fmt.Errorf("title is required")
	}
	result, err := sdk.CalendarTodos().Create(ctx, in.Title, in.Date)
	if err != nil {
		return nil, nil, toolError(err)
	}
	return nil, result, nil
}

type todoIDInput struct {
	ID int64 `json:"id" jsonschema:"numeric todo id"`
}

func completeTodoHandler(ctx context.Context, _ *mcp.CallToolRequest, in todoIDInput) (*mcp.CallToolResult, any, error) {
	if in.ID == 0 {
		return nil, nil, fmt.Errorf("id is required")
	}
	result, err := sdk.CalendarTodos().Complete(ctx, in.ID)
	if err != nil {
		return nil, nil, toolError(err)
	}
	return nil, result, nil
}

func uncompleteTodoHandler(ctx context.Context, _ *mcp.CallToolRequest, in todoIDInput) (*mcp.CallToolResult, any, error) {
	if in.ID == 0 {
		return nil, nil, fmt.Errorf("id is required")
	}
	result, err := sdk.CalendarTodos().Uncomplete(ctx, in.ID)
	if err != nil {
		return nil, nil, toolError(err)
	}
	return nil, result, nil
}

func deleteTodoHandler(ctx context.Context, _ *mcp.CallToolRequest, in todoIDInput) (*mcp.CallToolResult, any, error) {
	if in.ID == 0 {
		return nil, nil, fmt.Errorf("id is required")
	}
	if err := sdk.CalendarTodos().Delete(ctx, in.ID); err != nil {
		return nil, nil, toolError(err)
	}
	return nil, map[string]any{"deleted": true, "id": in.ID}, nil
}

// --- habits ---

type habitInput struct {
	ID   int64  `json:"id" jsonschema:"numeric habit id"`
	Date string `json:"date,omitempty" jsonschema:"date in YYYY-MM-DD (default: today)"`
}

func completeHabitHandler(ctx context.Context, _ *mcp.CallToolRequest, in habitInput) (*mcp.CallToolResult, any, error) {
	if in.ID == 0 {
		return nil, nil, fmt.Errorf("id is required")
	}
	date := in.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	result, err := sdk.Habits().Complete(ctx, date, in.ID)
	if err != nil {
		return nil, nil, toolError(err)
	}
	return nil, result, nil
}

func uncompleteHabitHandler(ctx context.Context, _ *mcp.CallToolRequest, in habitInput) (*mcp.CallToolResult, any, error) {
	if in.ID == 0 {
		return nil, nil, fmt.Errorf("id is required")
	}
	date := in.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	result, err := sdk.Habits().Uncomplete(ctx, date, in.ID)
	if err != nil {
		return nil, nil, toolError(err)
	}
	return nil, result, nil
}

// --- time tracking ---

func startTimetrackHandler(ctx context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	result, err := sdk.TimeTracks().Start(ctx, generated.StartTimeTrackJSONRequestBody{})
	if err != nil {
		return nil, nil, toolError(err)
	}
	return nil, result, nil
}

func stopTimetrackHandler(ctx context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	track, err := sdk.TimeTracks().GetOngoing(ctx)
	if err != nil {
		return nil, nil, toolError(err)
	}
	if track == nil {
		return nil, nil, fmt.Errorf("no active time track")
	}
	if err := sdk.TimeTracks().Stop(ctx, track.Id); err != nil {
		return nil, nil, toolError(err)
	}
	return nil, map[string]any{"stopped": true, "id": track.Id}, nil
}

func currentTimetrackHandler(ctx context.Context, _ *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	track, err := sdk.TimeTracks().GetOngoing(ctx)
	if err != nil {
		return nil, nil, toolError(err)
	}
	if track == nil {
		return nil, map[string]any{"active": false}, nil
	}
	return nil, map[string]any{"active": true, "track": track}, nil
}

type listTimetracksInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"max number of time tracks to return (0 = no limit)"`
}

func listTimetracksHandler(ctx context.Context, _ *mcp.CallToolRequest, in listTimetracksInput) (*mcp.CallToolResult, any, error) {
	resp, err := listPersonalRecordings(ctx)
	if err != nil {
		return nil, nil, toolError(err)
	}
	tracks := filterRecordingsByType(resp, "Calendar::TimeTrack")
	if in.Limit > 0 && len(tracks) > in.Limit {
		tracks = tracks[:in.Limit]
	}
	return nil, tracks, nil
}

// --- journal ---

type readJournalInput struct {
	Date string `json:"date,omitempty" jsonschema:"date in YYYY-MM-DD (default: today)"`
}

func readJournalHandler(ctx context.Context, _ *mcp.CallToolRequest, in readJournalInput) (*mcp.CallToolResult, any, error) {
	date := in.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	content, err := sdk.Journal().GetContent(ctx, date)
	if err != nil {
		return nil, nil, toolError(err)
	}
	return nil, map[string]any{"date": date, "content": content}, nil
}

type writeJournalInput struct {
	Content string `json:"content" jsonschema:"journal content (plain text or HTML); replaces the existing entry for this date"`
	Date    string `json:"date,omitempty" jsonschema:"date in YYYY-MM-DD (default: today)"`
}

func writeJournalHandler(ctx context.Context, _ *mcp.CallToolRequest, in writeJournalInput) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(in.Content) == "" {
		return nil, nil, fmt.Errorf("content is required")
	}
	date := in.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	if err := sdk.Journal().Update(ctx, date, in.Content); err != nil {
		return nil, nil, toolError(err)
	}
	return nil, map[string]any{"saved": true, "date": date}, nil
}

func listJournalEntriesHandler(ctx context.Context, _ *mcp.CallToolRequest, in listTodosInput) (*mcp.CallToolResult, any, error) {
	resp, err := listPersonalRecordings(ctx)
	if err != nil {
		return nil, nil, toolError(err)
	}
	entries := filterRecordingsByType(resp, "Calendar::JournalEntry")
	if in.Limit > 0 && len(entries) > in.Limit {
		entries = entries[:in.Limit]
	}
	return nil, entries, nil
}

// --- registration ---

func registerCalendarTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_calendars",
		Description: "List the user's calendars with their ids, names, and ownership.",
	}, listCalendarsHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_recordings",
		Description: "List recordings (events, todos, journal entries, habits, time tracks) for a calendar within a date range. Defaults to today through 30 days ahead.",
	}, listRecordingsHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_todos",
		Description: "List todos from the user's personal calendar.",
	}, listTodosHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_todo",
		Description: "Create a todo on the user's personal calendar.",
	}, createTodoHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "complete_todo",
		Description: "Mark a todo as complete.",
	}, completeTodoHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "uncomplete_todo",
		Description: "Mark a previously completed todo as incomplete.",
	}, uncompleteTodoHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "delete_todo",
		Description: "Delete a todo permanently.",
	}, deleteTodoHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "complete_habit",
		Description: "Mark a habit as complete for a given date (default: today).",
	}, completeHabitHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "uncomplete_habit",
		Description: "Remove a habit completion for a given date (default: today).",
	}, uncompleteHabitHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "start_timetrack",
		Description: "Start a new time-tracking session.",
	}, startTimetrackHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "stop_timetrack",
		Description: "Stop the currently ongoing time-tracking session.",
	}, stopTimetrackHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "current_timetrack",
		Description: "Return the currently ongoing time-tracking session, if any.",
	}, currentTimetrackHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_timetracks",
		Description: "List time-tracking sessions from the user's personal calendar.",
	}, listTimetracksHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "read_journal",
		Description: "Read the journal entry for a given date (default: today). Returns the raw HTML/text content.",
	}, readJournalHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "write_journal",
		Description: "Create or replace the journal entry for a given date (default: today).",
	}, writeJournalHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_journal_entries",
		Description: "List journal entries from the user's personal calendar.",
	}, listJournalEntriesHandler)
}
