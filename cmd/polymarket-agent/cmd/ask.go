package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/grokify/polymarket-go/internal/llm"
	"github.com/plexusone/omnillm-core"
	"github.com/spf13/cobra"
)

var askCmd = &cobra.Command{
	Use:   "ask [prompt...]",
	Short: "Ask the LLM a question",
	Long: `Send an ad-hoc query to the LLM and display the response.

The prompt can be provided as command arguments or piped via stdin.

Examples:
  polymarket-agent ask "What are prediction markets?"
  polymarket-agent ask What is the capital of France
  echo "Explain Bitcoin" | polymarket-agent ask
  polymarket-agent ask --system "You are a helpful trading assistant" "What is a limit order?"

Requires ANTHROPIC_API_KEY environment variable.`,
	RunE: runAsk,
}

var (
	askSystemPrompt string
	askMaxTokens    int
	askStream       bool
)

func init() {
	rootCmd.AddCommand(askCmd)

	askCmd.Flags().StringVarP(&askSystemPrompt, "system", "s", "", "System prompt to use")
	askCmd.Flags().IntVar(&askMaxTokens, "max-tokens", 4096, "Maximum tokens in response")
	askCmd.Flags().BoolVar(&askStream, "stream", true, "Stream the response (default: true)")
}

func runAsk(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get prompt from args or stdin
	prompt, err := getPrompt(args, os.Stdin)
	if err != nil {
		return err
	}

	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
	}

	// Get model from root flag
	modelName, _ := cmd.Root().PersistentFlags().GetString("model")

	// Create omnillm client
	llmClient, err := omnillm.NewClient(omnillm.ClientConfig{
		Providers: []omnillm.ProviderConfig{
			{Provider: omnillm.ProviderNameAnthropic, APIKey: apiKey},
		},
	})
	if err != nil {
		return fmt.Errorf("creating LLM client: %w", err)
	}
	defer llmClient.Close()

	cfg := llm.AskConfig{
		Model:        modelName,
		SystemPrompt: askSystemPrompt,
		MaxTokens:    askMaxTokens,
	}

	if askStream {
		err = llm.AskStream(ctx, llmClient.Provider(), cfg, prompt, func(chunk string) error {
			fmt.Print(chunk)
			return nil
		})
		if err != nil {
			return err
		}
		fmt.Println() // Add newline after streaming
	} else {
		result, err := llm.Ask(ctx, llmClient.Provider(), cfg, prompt)
		if err != nil {
			return err
		}
		fmt.Println(result.Content)
	}

	return nil
}

// getPrompt retrieves the prompt from args or stdin.
// This is extracted for testability.
func getPrompt(args []string, stdin io.Reader) (string, error) {
	// Check if stdin has data
	if len(args) == 0 {
		// Try to read from stdin
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// Data is being piped
			scanner := bufio.NewScanner(stdin)
			var lines []string
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}
			if err := scanner.Err(); err != nil {
				return "", fmt.Errorf("reading stdin: %w", err)
			}
			if len(lines) > 0 {
				return strings.Join(lines, "\n"), nil
			}
		}
	}

	return llm.BuildPromptFromArgs(args)
}
