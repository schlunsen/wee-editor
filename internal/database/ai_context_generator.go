package database

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/schlunsen/claude-agent-sdk-go"
	"github.com/schlunsen/claude-agent-sdk-go/types"
)

// AIContextGenerator uses Claude to generate intelligent context prompts
type AIContextGenerator struct {
	apiKey        string
	model         string
	readmeContent string
	projectType   string
}

// NewAIContextGenerator creates a new AI context generator
func NewAIContextGenerator(apiKey, model, readmeContent, projectType string) *AIContextGenerator {
	if model == "" {
		model = "haiku"
	}

	// If no API key provided, try to get from environment
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("CLAUDE_API_KEY")
		}
	}

	return &AIContextGenerator{
		apiKey:        apiKey,
		model:         model,
		readmeContent: readmeContent,
		projectType:   projectType,
	}
}

// GenerateContextPrompt generates an intelligent context prompt using Claude
func (acg *AIContextGenerator) GenerateContextPrompt(
	ctx context.Context,
	areaName string,
	subdomainType string,
	relativePath string,
	existingDescription string,
) (string, error) {
	// Set the API key in environment for the Query function if provided
	// If empty, claude.Query will use Claude Desktop's authentication
	if acg.apiKey != "" {
		os.Setenv("ANTHROPIC_API_KEY", acg.apiKey)
		defer os.Unsetenv("ANTHROPIC_API_KEY") // Clean up
	}

	// Build the prompt for Claude
	prompt := acg.buildPrompt(areaName, subdomainType, relativePath, existingDescription)

	// Create options with the model
	opts := types.NewClaudeAgentOptions().
		WithModel(acg.model)

	// Create a timeout context
	genCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Query Claude using the Query function
	messages, err := claude.Query(genCtx, prompt, opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate context: %w", err)
	}

	// Extract the text from the response
	contextPrompt := acg.extractContextFromMessages(messages)
	if contextPrompt == "" {
		return "", fmt.Errorf("no response from Claude")
	}

	return contextPrompt, nil
}

// GenerateMultipleContexts generates context prompts for multiple areas
func (acg *AIContextGenerator) GenerateMultipleContexts(
	ctx context.Context,
	areas []struct {
		Name          string
		SubdomainType string
		RelativePath  string
		Description   string
	},
) (map[string]string, error) {
	results := make(map[string]string)

	for _, area := range areas {
		contextPrompt, err := acg.GenerateContextPrompt(
			ctx,
			area.Name,
			area.SubdomainType,
			area.RelativePath,
			area.Description,
		)
		if err != nil {
			// Log error but continue with other areas
			fmt.Printf("failed to generate context for %s: %v\n", area.Name, err)
			continue
		}
		results[area.Name] = contextPrompt
	}

	return results, nil
}

// buildPrompt constructs the prompt for Claude
func (acg *AIContextGenerator) buildPrompt(
	areaName string,
	subdomainType string,
	relativePath string,
	existingDescription string,
) string {
	var sb strings.Builder

	sb.WriteString("You are a code context generation expert. Generate a concise, actionable context prompt for a developer working on a specific project area.\n\n")

	sb.WriteString("Project Information:\n")
	if acg.projectType != "" {
		sb.WriteString(fmt.Sprintf("- Project Type: %s\n", acg.projectType))
	}
	if acg.readmeContent != "" {
		readmePreview := acg.truncateText(acg.readmeContent, 500)
		sb.WriteString(fmt.Sprintf("- Project README Preview:\n%s\n\n", readmePreview))
	}

	sb.WriteString("Area Information:\n")
	sb.WriteString(fmt.Sprintf("- Area Name: %s\n", areaName))
	sb.WriteString(fmt.Sprintf("- Area Type: %s\n", subdomainType))
	sb.WriteString(fmt.Sprintf("- Relative Path: %s\n", relativePath))
	if existingDescription != "" {
		sb.WriteString(fmt.Sprintf("- Auto-Generated Description: %s\n", existingDescription))
	}

	sb.WriteString("\n")
	sb.WriteString("Requirements:\n")
	sb.WriteString("1. Generate a context prompt that a developer would use when working in this area\n")
	sb.WriteString("2. The prompt should be specific to the project type and area\n")
	sb.WriteString("3. Include best practices relevant to the technology stack\n")
	sb.WriteString("4. Be concise (2-3 sentences max)\n")
	sb.WriteString("5. Focus on actionable guidance\n")
	sb.WriteString("6. Return ONLY the context prompt, no additional text or explanations\n\n")

	sb.WriteString("Generate the context prompt now:")

	return sb.String()
}

// extractContextFromMessages extracts text content from Claude's response
func (acg *AIContextGenerator) extractContextFromMessages(messages <-chan types.Message) string {
	var result strings.Builder

	for msg := range messages {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			for _, content := range m.Content {
				if textBlock, ok := content.(*types.TextBlock); ok {
					result.WriteString(textBlock.Text)
				}
			}
		case *types.ResultMessage:
			// End of response
			break
		}
	}

	return strings.TrimSpace(result.String())
}

// truncateText truncates text to a maximum length
func (acg *AIContextGenerator) truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength] + "..."
}

// AnalyzeProjectStructure analyzes the project structure and README to understand the project better
func (acg *AIContextGenerator) AnalyzeProjectStructure(
	ctx context.Context,
	projectPath string,
	fileStructure string,
) (string, error) {
	// Set the API key in environment if provided
	// If empty, claude.Query will use Claude Desktop's authentication
	if acg.apiKey != "" {
		os.Setenv("ANTHROPIC_API_KEY", acg.apiKey)
		defer os.Unsetenv("ANTHROPIC_API_KEY")
	}

	prompt := fmt.Sprintf(`Analyze this project structure and provide a brief summary of the project's main purpose and architecture:

Project Path: %s

File Structure:
%s

Provide your analysis in 2-3 sentences, focusing on:
1. Main purpose of the project
2. Key components/layers
3. Primary technologies

Be concise and return only the analysis.`, projectPath, fileStructure)

	opts := types.NewClaudeAgentOptions().
		WithModel(acg.model)

	genCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	messages, err := claude.Query(genCtx, prompt, opts)
	if err != nil {
		return "", fmt.Errorf("failed to analyze project: %w", err)
	}

	analysis := acg.extractContextFromMessages(messages)
	if analysis == "" {
		return "", fmt.Errorf("no analysis from Claude")
	}

	return analysis, nil
}

// SuggestFilePatterns uses Claude to suggest appropriate file patterns for an area
func (acg *AIContextGenerator) SuggestFilePatterns(
	ctx context.Context,
	areaName string,
	subdomainType string,
) ([]string, error) {
	// Set the API key in environment if provided
	// If empty, claude.Query will use Claude Desktop's authentication
	if acg.apiKey != "" {
		os.Setenv("ANTHROPIC_API_KEY", acg.apiKey)
		defer os.Unsetenv("ANTHROPIC_API_KEY")
	}

	prompt := fmt.Sprintf(`You are a file pattern expert. For the following project area, suggest glob patterns for files typically in this area:

Project Type: %s
Area Name: %s
Area Type: %s

Return ONLY a newline-separated list of glob patterns. Examples:
**/*.vue
**/*.ts
src/components/**

Suggested patterns:`, acg.projectType, areaName, subdomainType)

	opts := types.NewClaudeAgentOptions().
		WithModel(acg.model)

	genCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	messages, err := claude.Query(genCtx, prompt, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest patterns: %w", err)
	}

	response := acg.extractContextFromMessages(messages)
	if response == "" {
		return nil, fmt.Errorf("no patterns from Claude")
	}

	// Parse the response into a list of patterns
	var patterns []string
	for _, line := range strings.Split(response, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "//") {
			patterns = append(patterns, line)
		}
	}

	return patterns, nil
}
