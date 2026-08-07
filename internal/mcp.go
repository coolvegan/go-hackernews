package internal

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type HackernewsMcp struct {
	s    *server.MCPServer
	data []*Item
}

func RunHackernewsMcp(data *[]*Item, lock *sync.RWMutex) *HackernewsMcp {
	s := server.NewMCPServer(
		"Hackernews Server",
		"0.0.1",
		server.WithTaskCapabilities(true, true, true),
		server.WithMaxConcurrentTasks(10), // Allow up to 10 concurrent running tasks
	)
	initialize(s, data, lock)
	return &HackernewsMcp{s: s}
}

func initialize(s *server.MCPServer, data *[]*Item, lock *sync.RWMutex) {
	hackernewsTool := mcp.NewTool("hackernews",
		mcp.WithDescription("See the newest entries / articles at hackernews"),
		mcp.WithString("filter",
			mcp.Required(),
			mcp.Description("Filter the data. (full, summary"), mcp.Enum("full", "summary")),
	)
	hackernewsToolCount := mcp.NewTool("hackernewsArticleCount",
		mcp.WithDescription("See the available article count of hacker news"),
	)
	s.AddTool(hackernewsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		lock.RLock()
		res, err := json.Marshal(*data)
		lock.RUnlock()
		if err != nil {
			return nil, err
		}
		filter, err := request.RequireString("filter")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		switch filter {
		case "summary":
			summary := make([]ArticleSummary, 0, len(*data))
			for _, d := range *data {
				summary = append(summary, ArticleSummary{Title: d.Title, Text: d.Text, Url: d.Url, Score: d.Score})
			}
			res, err := json.Marshal(summary)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(res)), nil
		case "full":
			return mcp.NewToolResultText(string(res)), nil
		}
		return nil, err

	})
	s.AddTool(hackernewsToolCount, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		lock.RLock()
		res, err := json.Marshal(len(*data))
		lock.RUnlock()
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(res)), nil
	})
	sseServer := server.NewSSEServer(s, server.WithBaseURL("http://localhost:13333"))

	log.Println("MCP SSE-Server läuft unter http://localhost:13333/sse")

	// 2. Start() registriert die Routen UND startet den HTTP-Server auf dem Port
	if err := sseServer.Start(":13333"); err != nil {
		log.Fatalf("Server-Fehler: %v", err)
	}
}
