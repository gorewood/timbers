package main

import (
	"context"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorewood/timbers/internal/llm"
	"github.com/gorewood/timbers/internal/output"
	"github.com/spf13/cobra"
)

// generateFlags holds all flag values for the generate command.
type generateFlags struct {
	model       string
	provider    string
	system      string
	input       string
	temperature float64
	maxTokens   int
	timeout     int
}

// newGenerateCmd creates the generate command.
func newGenerateCmd() *cobra.Command {
	var flags generateFlags

	cmd := &cobra.Command{
		Use:   "generate [prompt]",
		Short: "Generate LLM completions",
		Long: `Generate completions using LLM providers (Anthropic, OpenAI, Google, Local).

This is a composable primitive for piping text through an LLM.
Defaults to local LLM server if no model specified.

Examples:
  # Use local LLM (default)
  timbers generate "Explain recursion"

  # Use cloud providers
  timbers generate "Explain recursion" --model claude-haiku
  timbers generate "Explain recursion" --model gemini-flash
  timbers generate "Explain recursion" --model openai-nano

  # Pipe input through stdin
  echo "Summarize this" | timbers generate

  # With system prompt
  timbers generate "Write tests" --model claude-sonnet --system "You are a Go expert"

  # JSON output
  timbers generate "List 3 items" --json

Model shortcuts:
  Anthropic: haiku, sonnet, opus (or claude-haiku, claude-sonnet, claude-opus)
  OpenAI:    nano, mini, gpt-5 (or openai-nano, openai-mini)
  Google:    flash, flash-lite, pro (or gemini-flash, gemini-pro)
  Local:     local (default - uses loaded model in LM Studio/Ollama)

Environment variables:
  ANTHROPIC_API_KEY  Required for Anthropic models
  OPENAI_API_KEY     Required for OpenAI models
  GOOGLE_API_KEY     Required for Google models
  LOCAL_LLM_URL      Local server URL (default: http://localhost:1234/v1)`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(cmd, args, flags)
		},
	}

	cmd.Flags().StringVarP(&flags.model, "model", "m", "local", "Model name (default: local)")
	cmd.Flags().StringVarP(&flags.provider, "provider", "p", "", "Provider (anthropic, openai, google, local) - inferred if omitted")
	cmd.Flags().StringVarP(&flags.system, "system", "s", "", "System prompt")
	cmd.Flags().StringVarP(&flags.input, "input", "i", "", "Input file (default: stdin if no prompt argument)")
	cmd.Flags().Float64Var(&flags.temperature, "temperature", 0, "Temperature (0.0-1.0, 0 uses model default)")
	cmd.Flags().IntVar(&flags.maxTokens, "max-tokens", 0, "Max tokens to generate (0 uses model default)")
	cmd.Flags().IntVar(&flags.timeout, "timeout", 120, "Request timeout in seconds")

	return cmd
}

// validateGenerateFlags validates the LLM-related flags.
func validateGenerateFlags(flags generateFlags) error {
	if flags.temperature < 0 || flags.temperature > 2 {
		return output.NewUserError("temperature must be between 0 and 2, got " + formatFloat(flags.temperature))
	}
	if flags.timeout <= 0 {
		return output.NewUserError("timeout must be positive, got " + formatInt(flags.timeout))
	}
	if flags.maxTokens < 0 {
		return output.NewUserError("max-tokens must be non-negative, got " + formatInt(flags.maxTokens))
	}
	return nil
}

// formatFloat formats a float64 for error messages.
func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// formatInt formats an int for error messages.
func formatInt(i int) string {
	return strconv.Itoa(i)
}

// runGenerate executes the generate command.
func runGenerate(cmd *cobra.Command, args []string, flags generateFlags) error {
	printer := output.NewPrinter(cmd.OutOrStdout(), jsonFlag, output.IsTTY(cmd.OutOrStdout()))

	// Validate flags before any other work
	if err := validateGenerateFlags(flags); err != nil {
		printer.Error(err)
		return err
	}

	// Build prompt from args and/or stdin
	promptText, err := buildPromptFromSources(cmd, args, flags.input)
	if err != nil {
		printer.Error(err)
		return err
	}

	if promptText == "" {
		err := output.NewUserError("no prompt provided. Use argument, --input file, or pipe via stdin")
		printer.Error(err)
		return err
	}

	// Create LLM client
	client, err := llm.New(flags.model, llm.Provider(flags.provider))
	if err != nil {
		userErr := output.NewUserError(err.Error())
		printer.Error(userErr)
		return userErr
	}

	// Build request
	req := llm.Request{
		System:      flags.system,
		Prompt:      promptText,
		Temperature: flags.temperature,
		MaxTokens:   flags.maxTokens,
	}

	// Execute with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(flags.timeout)*time.Second)
	defer cancel()

	resp, err := client.Complete(ctx, req)
	if err != nil {
		sysErr := output.NewSystemErrorWithCause("generation failed", err)
		printer.Error(sysErr)
		return sysErr
	}

	// Output result
	if jsonFlag {
		return printer.Success(map[string]any{
			"model":   resp.Model,
			"content": resp.Content,
		})
	}

	// Plain text output for piping
	printer.Print("%s\n", resp.Content)
	return nil
}

// buildPromptFromSources builds the prompt from args, stdin, and/or input file.
func buildPromptFromSources(cmd *cobra.Command, args []string, inputFile string) (string, error) {
	var parts []string

	// Add prompt argument if provided
	if len(args) > 0 && args[0] != "" {
		parts = append(parts, args[0])
	}

	// Add file content if specified
	if inputFile != "" {
		content, err := readInputFile(inputFile)
		if err != nil {
			return "", err
		}
		parts = append(parts, content)
	}

	// Add stdin content if available
	stdinContent, err := readStdinIfPiped(cmd)
	if err != nil {
		return "", err
	}
	if stdinContent != "" {
		parts = append(parts, stdinContent)
	}

	return strings.Join(parts, "\n\n"), nil
}

// readInputFile reads content from a file.
func readInputFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", output.NewUserError("failed to read input file: " + err.Error())
	}
	return string(content), nil
}

// readStdinIfPiped reads stdin content if it's piped (not a terminal).
func readStdinIfPiped(cmd *cobra.Command) (string, error) {
	stdin := cmd.InOrStdin()
	file, ok := stdin.(*os.File)
	if !ok {
		return "", nil
	}

	stat, err := file.Stat()
	if err != nil {
		// Can't stat stdin - assume it's not piped
		return "", nil //nolint:nilerr // stat failure means stdin isn't usable, not an error
	}

	// Check if stdin is piped (not a character device)
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return "", nil
	}

	content, err := io.ReadAll(stdin)
	if err != nil {
		return "", output.NewSystemErrorWithCause("failed to read stdin", err)
	}

	return strings.TrimSpace(string(content)), nil
}
