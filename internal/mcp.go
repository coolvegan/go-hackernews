package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type HackernewsMcp struct {
	s    *server.MCPServer
	data []*Item
}

func RunHackernewsMcp(mcpserver string, hackerNewsItemsMap *map[int]*Item, lock *sync.RWMutex) *HackernewsMcp {
	s := server.NewMCPServer(
		"Hackernews Server",
		"0.0.1",
		server.WithTaskCapabilities(true, true, true),
		server.WithMaxConcurrentTasks(10), // Allow up to 10 concurrent running tasks
	)
	initialize(s, mcpserver, hackerNewsItemsMap, lock)
	return &HackernewsMcp{s: s}
}

func initialize(s *server.MCPServer, mcpserver string, hackerNewsItemsMap *map[int]*Item, lock *sync.RWMutex) {
	hackernewsTool := mcp.NewTool("hackernews",
		mcp.WithDescription("See the newest entries / articles at hackernews"),
		mcp.WithString("filter",
			mcp.Required(),
			mcp.Description("Filter the data. (full, summary"), mcp.Enum("full", "summary")),
		mcp.WithNumber("minScore",
			mcp.Description("Min-Score; Article with lower Scores get filtered out")),
		mcp.WithNumber("maxAgeMinutes",
			mcp.Description("Max Age in Minutes; Older Articles are filtered out")),
	)
	hackernewsToolCount := mcp.NewTool("hackernewsArticleCount",
		mcp.WithDescription("See the available article count of hacker news"),
	)
	s.AddTool(hackernewsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filter, err := request.RequireString("filter")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		minScore := request.GetInt("minScore", 0)
		maxAgeMinutes := request.GetInt("maxAgeMinutes", 0)
		lock.RLock()
		filtered := make([]*Item, 0, len(*hackerNewsItemsMap))
		now := time.Now()
		for _, d := range *hackerNewsItemsMap {
			if d.Score < minScore {
				continue
			}
			if maxAgeMinutes > 0 && now.Sub(d.Created()) > time.Duration(maxAgeMinutes)*time.Minute {
				continue
			}
			filtered = append(filtered, d)
		}
		lock.RUnlock()

		switch filter {
		case "summary":
			summary := make([]ArticleSummary, 0, len(filtered))
			for _, d := range filtered {
				summary = append(summary, ArticleSummary{Title: d.Title, Text: d.Text, Url: d.Url, Score: d.Score})
			}
			res, err := json.Marshal(summary)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(res)), nil
		case "full":
			res, err := json.Marshal(filtered)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(res)), nil
		default:
			return mcp.NewToolResultError("unknown filter: " + filter), nil
		}
	})
	s.AddTool(hackernewsToolCount, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		lock.RLock()
		res, err := json.Marshal(len(*hackerNewsItemsMap))
		lock.RUnlock()
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(res)), nil
	})
	sseServer := server.NewSSEServer(s, server.WithBaseURL(fmt.Sprintf("%s", mcpserver)))

	log.Printf("MCP SSE-Server is running on %s\n", mcpserver)

	// 2. Start() registriert die Routen UND startet den HTTP-Server auf dem Port
	if err := sseServer.Start(fmt.Sprintf("%s", mcpserver)); err != nil {
		log.Fatalf("Server-Error: %v", err)
	}
}
