package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	ghErrors "github.com/github/github-mcp-server/pkg/errors"
	"github.com/github/github-mcp-server/pkg/translations"
	"github.com/github/github-mcp-server/pkg/utils"
	"github.com/google/go-github/v79/github"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// IssueDependency represents a dependency between issues
type IssueDependency struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	State   string `json:"state"`
	HTMLURL string `json:"html_url"`
}

// IssueDependenciesResponse represents the response from the dependencies API
type IssueDependenciesResponse struct {
	Dependencies []IssueDependency `json:"dependencies"`
}

// DependencyRequest represents the request body for adding or removing a dependency
type DependencyRequest struct {
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	IssueNumber int    `json:"issue_number"`
}

// ListBlockedBy creates a tool to list issues that a given issue is blocked by
func ListBlockedBy(getClient GetClientFn, t translations.TranslationHelperFunc) (mcp.Tool, mcp.ToolHandlerFor[map[string]any, any]) {
	schema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"owner": {
				Type:        "string",
				Description: "Repository owner (username or organization)",
			},
			"repo": {
				Type:        "string",
				Description: "Repository name",
			},
			"issue_number": {
				Type:        "number",
				Description: "The number of the issue",
			},
		},
		Required: []string{"owner", "repo", "issue_number"},
	}
	WithPagination(schema)

	return mcp.Tool{
			Name:        "issue_dependencies.list_blocked_by",
			Description: t("TOOL_ISSUE_DEPENDENCIES_LIST_BLOCKED_BY_DESCRIPTION", "List issues that a given issue is blocked by."),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_ISSUE_DEPENDENCIES_LIST_BLOCKED_BY_TITLE", "List blocking issues"),
				ReadOnlyHint: true,
			},
			InputSchema: schema,
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			owner, err := RequiredParam[string](args, "owner")
			if err != nil {
				return utils.NewToolResultError(err.Error()), nil, nil
			}

			repo, err := RequiredParam[string](args, "repo")
			if err != nil {
				return utils.NewToolResultError(err.Error()), nil, nil
			}

			issueNumber, err := RequiredInt(args, "issue_number")
			if err != nil {
				return utils.NewToolResultError(err.Error()), nil, nil
			}

			pagination, err := OptionalPaginationParams(args)
			if err != nil {
				return utils.NewToolResultError(err.Error()), nil, nil
			}

			client, err := getClient(ctx)
			if err != nil {
				return utils.NewToolResultErrorFromErr("failed to get GitHub client", err), nil, nil
			}

			result, err := listIssueDependencies(ctx, client, owner, repo, issueNumber, "blocked_by", pagination)
			return result, nil, err
		}
}

// ListBlocking creates a tool to list issues that a given issue is blocking
func ListBlocking(getClient GetClientFn, t translations.TranslationHelperFunc) (mcp.Tool, mcp.ToolHandlerFor[map[string]any, any]) {
	schema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"owner": {
				Type:        "string",
				Description: "Repository owner (username or organization)",
			},
			"repo": {
				Type:        "string",
				Description: "Repository name",
			},
			"issue_number": {
				Type:        "number",
				Description: "The number of the issue",
			},
		},
		Required: []string{"owner", "repo", "issue_number"},
	}
	WithPagination(schema)

	return mcp.Tool{
			Name:        "issue_dependencies.list_blocking",
			Description: t("TOOL_ISSUE_DEPENDENCIES_LIST_BLOCKING_DESCRIPTION", "List issues that a given issue is blocking."),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_ISSUE_DEPENDENCIES_LIST_BLOCKING_TITLE", "List blocked issues"),
				ReadOnlyHint: true,
			},
			InputSchema: schema,
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			owner, err := RequiredParam[string](args, "owner")
			if err != nil {
				return utils.NewToolResultError(err.Error()), nil, nil
			}

			repo, err := RequiredParam[string](args, "repo")
			if err != nil {
				return utils.NewToolResultError(err.Error()), nil, nil
			}

			issueNumber, err := RequiredInt(args, "issue_number")
			if err != nil {
				return utils.NewToolResultError(err.Error()), nil, nil
			}

			pagination, err := OptionalPaginationParams(args)
			if err != nil {
				return utils.NewToolResultError(err.Error()), nil, nil
			}

			client, err := getClient(ctx)
			if err != nil {
				return utils.NewToolResultErrorFromErr("failed to get GitHub client", err), nil, nil
			}

			result, err := listIssueDependencies(ctx, client, owner, repo, issueNumber, "blocking", pagination)
			return result, nil, err
		}
}

// AddBlockedBy creates a tool to add a blocked-by dependency
func AddBlockedBy(getClient GetClientFn, t translations.TranslationHelperFunc) (mcp.Tool, mcp.ToolHandlerFor[map[string]any, any]) {
	schema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"owner": {
				Type:        "string",
				Description: "Repository owner (username or organization)",
			},
			"repo": {
				Type:        "string",
				Description: "Repository name",
			},
			"issue_number": {
				Type:        "number",
				Description: "The number of the issue that is blocked",
			},
			"blocked_by_owner": {
				Type:        "string",
				Description: "Repository owner of the blocking issue (defaults to same owner)",
			},
			"blocked_by_repo": {
				Type:        "string",
				Description: "Repository name of the blocking issue (defaults to same repo)",
			},
			"blocked_by_issue_number": {
				Type:        "number",
				Description: "The number of the issue that is blocking",
			},
		},
		Required: []string{"owner", "repo", "issue_number", "blocked_by_issue_number"},
	}

	return mcp.Tool{
			Name:        "issue_dependencies.add_blocked_by",
			Description: t("TOOL_ISSUE_DEPENDENCIES_ADD_BLOCKED_BY_DESCRIPTION", "Add a blocked-by dependency to an issue."),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_ISSUE_DEPENDENCIES_ADD_BLOCKED_BY_TITLE", "Add blocking dependency"),
				ReadOnlyHint: false,
			},
			InputSchema: schema,
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			owner, err := RequiredParam[string](args, "owner")
			if err != nil {
				return utils.NewToolResultError(err.Error()), nil, nil
			}

			repo, err := RequiredParam[string](args, "repo")
			if err != nil {
				return utils.NewToolResultError(err.Error()), nil, nil
			}

			issueNumber, err := RequiredInt(args, "issue_number")
			if err != nil {
				return utils.NewToolResultError(err.Error()), nil, nil
			}

			blockedByIssueNumber, err := RequiredInt(args, "blocked_by_issue_number")
			if err != nil {
				return utils.NewToolResultError(err.Error()), nil, nil
			}

			// Optional: allow cross-repo dependencies
			blockedByOwner, err := OptionalParam[string](args, "blocked_by_owner")
			if err != nil {
				return utils.NewToolResultError(err.Error()), nil, nil
			}
			if blockedByOwner == "" {
				blockedByOwner = owner
			}

			blockedByRepo, err := OptionalParam[string](args, "blocked_by_repo")
			if err != nil {
				return utils.NewToolResultError(err.Error()), nil, nil
			}
			if blockedByRepo == "" {
				blockedByRepo = repo
			}

			client, err := getClient(ctx)
			if err != nil {
				return utils.NewToolResultErrorFromErr("failed to get GitHub client", err), nil, nil
			}

			result, err := addIssueDependency(ctx, client, owner, repo, issueNumber, blockedByOwner, blockedByRepo, blockedByIssueNumber)
			return result, nil, err
		}
}

// RemoveBlockedBy creates a tool to remove a blocked-by dependency
func RemoveBlockedBy(getClient GetClientFn, t translations.TranslationHelperFunc) (mcp.Tool, mcp.ToolHandlerFor[map[string]any, any]) {
	schema := &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"owner": {
				Type:        "string",
				Description: "Repository owner (username or organization)",
			},
			"repo": {
				Type:        "string",
				Description: "Repository name",
			},
			"issue_number": {
				Type:        "number",
				Description: "The number of the issue that is blocked",
			},
			"blocked_by_owner": {
				Type:        "string",
				Description: "Repository owner of the blocking issue (defaults to same owner)",
			},
			"blocked_by_repo": {
				Type:        "string",
				Description: "Repository name of the blocking issue (defaults to same repo)",
			},
			"blocked_by_issue_number": {
				Type:        "number",
				Description: "The number of the issue that is blocking",
			},
		},
		Required: []string{"owner", "repo", "issue_number", "blocked_by_issue_number"},
	}

	return mcp.Tool{
			Name:        "issue_dependencies.remove_blocked_by",
			Description: t("TOOL_ISSUE_DEPENDENCIES_REMOVE_BLOCKED_BY_DESCRIPTION", "Remove a blocked-by dependency from an issue."),
			Annotations: &mcp.ToolAnnotations{
				Title:        t("TOOL_ISSUE_DEPENDENCIES_REMOVE_BLOCKED_BY_TITLE", "Remove blocking dependency"),
				ReadOnlyHint: false,
			},
			InputSchema: schema,
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
			owner, err := RequiredParam[string](args, "owner")
			if err != nil {
				return utils.NewToolResultError(err.Error()), nil, nil
			}

			repo, err := RequiredParam[string](args, "repo")
			if err != nil {
				return utils.NewToolResultError(err.Error()), nil, nil
			}

			issueNumber, err := RequiredInt(args, "issue_number")
			if err != nil {
				return utils.NewToolResultError(err.Error()), nil, nil
			}

			blockedByIssueNumber, err := RequiredInt(args, "blocked_by_issue_number")
			if err != nil {
				return utils.NewToolResultError(err.Error()), nil, nil
			}

			// Optional: allow cross-repo dependencies
			blockedByOwner, err := OptionalParam[string](args, "blocked_by_owner")
			if err != nil {
				return utils.NewToolResultError(err.Error()), nil, nil
			}
			if blockedByOwner == "" {
				blockedByOwner = owner
			}

			blockedByRepo, err := OptionalParam[string](args, "blocked_by_repo")
			if err != nil {
				return utils.NewToolResultError(err.Error()), nil, nil
			}
			if blockedByRepo == "" {
				blockedByRepo = repo
			}

			client, err := getClient(ctx)
			if err != nil {
				return utils.NewToolResultErrorFromErr("failed to get GitHub client", err), nil, nil
			}

			result, err := removeIssueDependency(ctx, client, owner, repo, issueNumber, blockedByOwner, blockedByRepo, blockedByIssueNumber)
			return result, nil, err
		}
}

// listIssueDependencies fetches dependencies for an issue
func listIssueDependencies(ctx context.Context, client *github.Client, owner, repo string, issueNumber int, dependencyType string, pagination PaginationParams) (*mcp.CallToolResult, error) {
	url := fmt.Sprintf("repos/%s/%s/issues/%d/dependencies/%s", owner, repo, issueNumber, dependencyType)

	// Add pagination parameters
	if pagination.Page > 0 || pagination.PerPage > 0 {
		url = fmt.Sprintf("%s?page=%d&per_page=%d", url, pagination.Page, pagination.PerPage)
	}

	req, err := client.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var deps IssueDependenciesResponse
	resp, err := client.Do(ctx, req, &deps)
	if err != nil {
		return ghErrors.NewGitHubAPIErrorResponse(ctx,
			fmt.Sprintf("failed to list %s dependencies", dependencyType),
			resp,
			err,
		), nil
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}
		return utils.NewToolResultError(fmt.Sprintf("failed to list dependencies: %s", string(body))), nil
	}

	r, err := json.Marshal(deps.Dependencies)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return utils.NewToolResultText(string(r)), nil
}

// addIssueDependency adds a blocked-by dependency
func addIssueDependency(ctx context.Context, client *github.Client, owner, repo string, issueNumber int, blockedByOwner, blockedByRepo string, blockedByIssueNumber int) (*mcp.CallToolResult, error) {
	url := fmt.Sprintf("repos/%s/%s/issues/%d/dependencies/blocked_by", owner, repo, issueNumber)

	reqBody := DependencyRequest{
		Owner:       blockedByOwner,
		Repo:        blockedByRepo,
		IssueNumber: blockedByIssueNumber,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := client.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var result map[string]any
	resp, err := client.Do(ctx, req, &result)
	if err != nil {
		return ghErrors.NewGitHubAPIErrorResponse(ctx,
			"failed to add blocked_by dependency",
			resp,
			err,
		), nil
	}
	defer func() { _ = resp.Body.Close() }()

	// Accept both 200 OK (when dependency already exists) and 201 Created (when newly created)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}
		return utils.NewToolResultError(fmt.Sprintf("failed to add dependency: %s", string(body))), nil
	}

	r, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return utils.NewToolResultText(string(r)), nil
}

// removeIssueDependency removes a blocked-by dependency
func removeIssueDependency(ctx context.Context, client *github.Client, owner, repo string, issueNumber int, blockedByOwner, blockedByRepo string, blockedByIssueNumber int) (*mcp.CallToolResult, error) {
	url := fmt.Sprintf("repos/%s/%s/issues/%d/dependencies/blocked_by", owner, repo, issueNumber)

	reqBody := DependencyRequest{
		Owner:       blockedByOwner,
		Repo:        blockedByRepo,
		IssueNumber: blockedByIssueNumber,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := client.NewRequest("DELETE", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var result map[string]any
	resp, err := client.Do(ctx, req, &result)
	if err != nil {
		return ghErrors.NewGitHubAPIErrorResponse(ctx,
			"failed to remove blocked_by dependency",
			resp,
			err,
		), nil
	}
	defer func() { _ = resp.Body.Close() }()

	// Accept both 200 OK (when response has content) and 204 No Content (when no response body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}
		return utils.NewToolResultError(fmt.Sprintf("failed to remove dependency: %s", string(body))), nil
	}

	r, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return utils.NewToolResultText(string(r)), nil
}
