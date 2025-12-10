package github

import (
	"context"
	"net/http"
	"testing"

	"github.com/github/github-mcp-server/internal/toolsnaps"
	"github.com/github/github-mcp-server/pkg/translations"
	"github.com/google/go-github/v79/github"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ListBlockedBy(t *testing.T) {
	// Verify tool definition once
	mockClient := github.NewClient(nil)
	tool, _ := ListBlockedBy(stubGetClientFn(mockClient), translations.NullTranslationHelper)
	require.NoError(t, toolsnaps.Test(tool.Name, tool))

	assert.Equal(t, "issue_dependencies.list_blocked_by", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.True(t, tool.Annotations.ReadOnlyHint)
	assert.Contains(t, tool.InputSchema.(*jsonschema.Schema).Properties, "owner")
	assert.Contains(t, tool.InputSchema.(*jsonschema.Schema).Properties, "repo")
	assert.Contains(t, tool.InputSchema.(*jsonschema.Schema).Properties, "issue_number")
	assert.ElementsMatch(t, tool.InputSchema.(*jsonschema.Schema).Required, []string{"owner", "repo", "issue_number"})

	tests := []struct {
		name               string
		mockedClient       *http.Client
		requestArgs        map[string]interface{}
		expectHandlerError bool
		expectResultError  bool
		expectedErrMsg     string
	}{
		{
			name: "successful dependencies retrieval",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.EndpointPattern{
						Pattern: "/repos/owner/repo/issues/42/dependencies/blocked_by",
						Method:  "GET",
					},
					mockResponse(t, http.StatusOK, `{
						"dependencies": [
							{
								"number": 10,
								"title": "Blocking Issue",
								"state": "open",
								"html_url": "https://github.com/owner/repo/issues/10"
							}
						]
					}`),
				),
			),
			requestArgs: map[string]interface{}{
				"owner":        "owner",
				"repo":         "repo",
				"issue_number": float64(42),
			},
		},
		{
			name: "missing owner parameter",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatch(
					mock.EndpointPattern{
						Pattern: "/repos/owner/repo/issues/42/dependencies/blocked_by",
						Method:  "GET",
					},
					nil,
				),
			),
			requestArgs: map[string]interface{}{
				"repo":         "repo",
				"issue_number": float64(42),
			},
			expectResultError: true,
			expectedErrMsg:    "missing required parameter: owner",
		},
		{
			name: "missing repo parameter",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatch(
					mock.EndpointPattern{
						Pattern: "/repos/owner/repo/issues/42/dependencies/blocked_by",
						Method:  "GET",
					},
					nil,
				),
			),
			requestArgs: map[string]interface{}{
				"owner":        "owner",
				"issue_number": float64(42),
			},
			expectResultError: true,
			expectedErrMsg:    "missing required parameter: repo",
		},
		{
			name: "missing issue_number parameter",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatch(
					mock.EndpointPattern{
						Pattern: "/repos/owner/repo/issues/42/dependencies/blocked_by",
						Method:  "GET",
					},
					nil,
				),
			),
			requestArgs: map[string]interface{}{
				"owner": "owner",
				"repo":  "repo",
			},
			expectResultError: true,
			expectedErrMsg:    "missing required parameter: issue_number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := github.NewClient(tt.mockedClient)
			_, handler := ListBlockedBy(stubGetClientFn(client), translations.NullTranslationHelper)

			result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, tt.requestArgs)

			if tt.expectHandlerError {
				require.Error(t, err)
				if tt.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
			} else {
				require.NoError(t, err)
			}

			if tt.expectResultError {
				require.NotNil(t, result)
				assert.True(t, result.IsError)
				if tt.expectedErrMsg != "" {
					textContent := getErrorResult(t, result)
					assert.Contains(t, textContent.Text, tt.expectedErrMsg)
				}
			}
		})
	}
}

func Test_ListBlocking(t *testing.T) {
	// Verify tool definition once
	mockClient := github.NewClient(nil)
	tool, _ := ListBlocking(stubGetClientFn(mockClient), translations.NullTranslationHelper)
	require.NoError(t, toolsnaps.Test(tool.Name, tool))

	assert.Equal(t, "issue_dependencies.list_blocking", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.True(t, tool.Annotations.ReadOnlyHint)
	assert.Contains(t, tool.InputSchema.(*jsonschema.Schema).Properties, "owner")
	assert.Contains(t, tool.InputSchema.(*jsonschema.Schema).Properties, "repo")
	assert.Contains(t, tool.InputSchema.(*jsonschema.Schema).Properties, "issue_number")
	assert.ElementsMatch(t, tool.InputSchema.(*jsonschema.Schema).Required, []string{"owner", "repo", "issue_number"})
}

func Test_AddBlockedBy(t *testing.T) {
	// Verify tool definition once
	mockClient := github.NewClient(nil)
	tool, _ := AddBlockedBy(stubGetClientFn(mockClient), translations.NullTranslationHelper)
	require.NoError(t, toolsnaps.Test(tool.Name, tool))

	assert.Equal(t, "issue_dependencies.add_blocked_by", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.False(t, tool.Annotations.ReadOnlyHint)
	assert.Contains(t, tool.InputSchema.(*jsonschema.Schema).Properties, "owner")
	assert.Contains(t, tool.InputSchema.(*jsonschema.Schema).Properties, "repo")
	assert.Contains(t, tool.InputSchema.(*jsonschema.Schema).Properties, "issue_number")
	assert.Contains(t, tool.InputSchema.(*jsonschema.Schema).Properties, "blocked_by_issue_number")
	assert.ElementsMatch(t, tool.InputSchema.(*jsonschema.Schema).Required, []string{"owner", "repo", "issue_number", "blocked_by_issue_number"})

	tests := []struct {
		name               string
		mockedClient       *http.Client
		requestArgs        map[string]interface{}
		expectHandlerError bool
		expectResultError  bool
		expectedErrMsg     string
	}{
		{
			name: "successful dependency addition",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatchHandler(
					mock.EndpointPattern{
						Pattern: "/repos/owner/repo/issues/42/dependencies/blocked_by",
						Method:  "POST",
					},
					mockResponse(t, http.StatusOK, `{"success": true}`),
				),
			),
			requestArgs: map[string]interface{}{
				"owner":                   "owner",
				"repo":                    "repo",
				"issue_number":            float64(42),
				"blocked_by_issue_number": float64(10),
			},
		},
		{
			name: "missing blocked_by_issue_number parameter",
			mockedClient: mock.NewMockedHTTPClient(
				mock.WithRequestMatch(
					mock.EndpointPattern{
						Pattern: "/repos/owner/repo/issues/42/dependencies/blocked_by",
						Method:  "POST",
					},
					nil,
				),
			),
			requestArgs: map[string]interface{}{
				"owner":        "owner",
				"repo":         "repo",
				"issue_number": float64(42),
			},
			expectResultError: true,
			expectedErrMsg:    "missing required parameter: blocked_by_issue_number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := github.NewClient(tt.mockedClient)
			_, handler := AddBlockedBy(stubGetClientFn(client), translations.NullTranslationHelper)

			result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, tt.requestArgs)

			if tt.expectHandlerError {
				require.Error(t, err)
				if tt.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
			} else {
				require.NoError(t, err)
			}

			if tt.expectResultError {
				require.NotNil(t, result)
				assert.True(t, result.IsError)
				if tt.expectedErrMsg != "" {
					textContent := getErrorResult(t, result)
					assert.Contains(t, textContent.Text, tt.expectedErrMsg)
				}
			}
		})
	}
}

func Test_RemoveBlockedBy(t *testing.T) {
	// Verify tool definition once
	mockClient := github.NewClient(nil)
	tool, _ := RemoveBlockedBy(stubGetClientFn(mockClient), translations.NullTranslationHelper)
	require.NoError(t, toolsnaps.Test(tool.Name, tool))

	assert.Equal(t, "issue_dependencies.remove_blocked_by", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.False(t, tool.Annotations.ReadOnlyHint)
	assert.Contains(t, tool.InputSchema.(*jsonschema.Schema).Properties, "owner")
	assert.Contains(t, tool.InputSchema.(*jsonschema.Schema).Properties, "repo")
	assert.Contains(t, tool.InputSchema.(*jsonschema.Schema).Properties, "issue_number")
	assert.Contains(t, tool.InputSchema.(*jsonschema.Schema).Properties, "blocked_by_issue_number")
	assert.ElementsMatch(t, tool.InputSchema.(*jsonschema.Schema).Required, []string{"owner", "repo", "issue_number", "blocked_by_issue_number"})
}
