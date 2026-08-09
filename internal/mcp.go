package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type HackernewsMcp struct {
	s    *server.MCPServer
	data []*Item
}

func RunHackernewsMcp(mcpserver string, hackerNewsItemsMap HNData, lock *sync.RWMutex) *HackernewsMcp {
	s := server.NewMCPServer(
		"Hackernews Server",
		"0.0.1",
		server.WithTaskCapabilities(true, true, true),
		server.WithMaxConcurrentTasks(10), // Allow up to 10 concurrent running tasks
	)
	initialize(s, mcpserver, hackerNewsItemsMap, lock)
	return &HackernewsMcp{s: s}
}

func initialize(s *server.MCPServer, mcpserver string, hackerNewsItemsMap HNData, lock *sync.RWMutex) {
	hackernewsTool := mcp.NewTool("hackernews",
		mcp.WithDescription("See the newest entries / articles at hackernews"),
		mcp.WithString("filter",
			mcp.Required(),
			mcp.Description("Filter the data. (full, summary)"), mcp.Enum("full", "summary")),
		mcp.WithNumber("minScore",
			mcp.Description("Min-Score; Article with lower Scores get filtered out")),
		mcp.WithNumber("maxAgeMinutes",
			mcp.Description("Max Age in Minutes; Older Articles are filtered out"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Return at most N articles, highest score first"),
		),
		mcp.WithNumber("id",
			mcp.Description("Story id; returns that story's comment structure as parent-id -> []child ids (no text). Use the returned comment ids with the 'comments' parameter to fetch their text."),
		),
		mcp.WithArray("comments",
			mcp.WithNumberItems(),
			mcp.Description("Comment ids (NOT story ids) to fetch the comment text for. Get comment ids from the 'id' parameter's comment structure. Returns the text of each requested comment."),
		),
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
		limit := request.GetInt("limit", 0)

		// Comments-by-text: fetch the text for explicitly requested comment
		// ids. Comments live in their story's tree, so we scan all items once.
		if commentIds := request.GetIntSlice("comments", nil); len(commentIds) > 0 {
			want := make(map[int]struct{}, len(commentIds))
			for _, cid := range commentIds {
				want[cid] = struct{}{}
			}
			lock.RLock()
			found := make([]CommentView, 0, len(want))
			seen := make(map[int]struct{}, len(want))
			for _, item := range hackerNewsItemsMap {
				lookup := item.CommentLookup()
				for id := range want {
					if _, ok := seen[id]; ok {
						continue
					}
					if cv, ok := lookup[id]; ok {
						found = append(found, cv)
						seen[id] = struct{}{}
					}
				}
			}
			lock.RUnlock()
			res, err := json.Marshal(found)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(res)), nil
		}

		// Comments-by-id: return the story's comment structure as a
		// parent-id -> []child-ids map (no text). Text is fetched separately
		// via the comments parameter.
		if storyId := request.GetInt("id", 0); storyId > 0 {
			lock.RLock()
			item, ok := (hackerNewsItemsMap)[storyId]
			lock.RUnlock()
			if !ok {
				return mcp.NewToolResultError(fmt.Sprintf("story id %d not in memory", storyId)), nil
			}
			cm, count := item.CommentMap()
			av := ArticleView{
				Id:           item.Id,
				Score:        item.Score,
				Time:         item.Time,
				Title:        item.Title,
				Text:         item.Text,
				Type:         item.Type,
				By:           item.By,
				Url:          item.Url,
				Parent:       item.Parent,
				Dead:         item.Dead,
				CommentCount: count,
				Comments:     cm,
			}
			res, err := json.Marshal(av)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(res)), nil
		}

		lock.RLock()
		filtered := make([]*Item, 0, len(hackerNewsItemsMap))
		now := time.Now()
		for _, d := range hackerNewsItemsMap {
			if d.Score < minScore {
				continue
			}
			if maxAgeMinutes > 0 && now.Sub(d.Created()) > time.Duration(maxAgeMinutes)*time.Minute {
				continue
			}
			filtered = append(filtered, d)
		}
		lock.RUnlock()

		// Highest score first, then optionally limit the result set.
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Score > filtered[j].Score
		})
		if limit > 0 && len(filtered) > limit {
			filtered = filtered[:limit]
		}

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
			view := make([]ArticleView, 0, len(filtered))
			for _, d := range filtered {
				view = append(view, ArticleView{
					Id:           d.Id,
					Score:        d.Score,
					Time:         d.Time,
					Title:        d.Title,
					Text:         d.Text,
					Type:         d.Type,
					By:           d.By,
					Url:          d.Url,
					Parent:       d.Parent,
					Dead:         d.Dead,
					CommentCount: len(d.Comments),
				})
			}
			res, err := json.Marshal(view)
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
		res, err := json.Marshal(len(hackerNewsItemsMap))
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
