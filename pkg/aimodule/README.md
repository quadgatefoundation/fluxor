# AI Module

The AI module (`pkg/aimodule`) provides a production-ready AI integration for Fluxor workflows with multi-provider support, tool calling, caching, and rate limiting.

## Features

- **Multi-provider support**: OpenAI, Anthropic, Ollama, Grok, Gemini, and custom providers
- **Tool calling**: Full support for OpenAI function calling
- **Caching**: In-memory response caching with TTL
- **Rate limiting**: Token bucket rate limiter with per-minute and per-day limits
- **Workflow integration**: Seamless integration with Fluxor workflow engine
- **Template support**: Prompt templating with `{{ $.input.field }}` syntax
- **Retry logic**: Automatic retry with exponential backoff
- **Production ready**: Error handling, timeout management, and logging

## Quick Start

### Using in Workflows

```json
{
  "id": "ai-chat-workflow",
  "nodes": [
    {
      "id": "trigger",
      "type": "webhook"
    },
    {
      "id": "ai-chat",
      "type": "aimodule.chat",
      "config": {
        "provider": "openai",
        "model": "gpt-4o",
        "prompt": "Bạn là trợ lý hỗ trợ khách hàng MoMo. Trả lời: {{ $.input.query }}",
        "temperature": 0.7,
        "maxTokens": 500
      },
      "next": ["output"]
    },
    {
      "id": "output",
      "type": "respond"
    }
  ]
}
```

### Using Programmatically

```go
import (
    "context"
    "github.com/fluxorio/fluxor/pkg/aimodule"
)

// Create client
config := aimodule.Config{
    Provider:     aimodule.ProviderOpenAI,
    APIKey:       "your-api-key", // or use OPENAI_API_KEY env var
    DefaultModel: "gpt-4o",
    Timeout:      60 * time.Second,
    MaxRetries:   3,
}

client, err := aimodule.NewClient(config)
if err != nil {
    log.Fatal(err)
}

// Chat completion
req := aimodule.ChatRequest{
    Model: "gpt-4o",
    Messages: []aimodule.Message{
        {Role: "user", Content: "Hello, world!"},
    },
    Temperature: 0.7,
}

resp, err := client.Chat(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

fmt.Println(resp.Choices[0].Message.Content)
```

## Configuration

### Environment Variables

- `OPENAI_API_KEY` - OpenAI API key
- `ANTHROPIC_API_KEY` - Anthropic API key
- `GROK_API_KEY` - Grok API key
- `GEMINI_API_KEY` - Gemini API key

### Config Options

```go
type Config struct {
    Provider     Provider        // AI provider
    APIKey       string          // API key (or use env var)
    BaseURL      string          // Base URL (optional)
    DefaultModel string          // Default model
    Timeout      time.Duration  // Request timeout
    MaxRetries   int            // Max retry attempts
    RateLimit    *RateLimitConfig // Rate limiting
    Cache        *CacheConfig   // Response caching
}
```

## Workflow Nodes

### `aimodule.chat`

Chat completion node with template support.

**Config:**
- `provider` - Provider name (default: "openai")
- `model` - Model name
- `prompt` - Prompt template (supports `{{ $.input.field }}`)
- `messages` - Array of messages (alternative to prompt)
- `temperature` - Temperature (0-2, default: 1.0)
- `maxTokens` - Max tokens
- `tools` - Array of tool definitions for function calling
- `toolChoice` - Tool choice ("auto", "none", or function name)
- `responseField` - Output field name (default: "response")

### `aimodule.embed`

Embedding generation node.

**Config:**
- `provider` - Provider name (default: "openai")
- `model` - Embedding model (default: "text-embedding-ada-002")
- `input` - Text or array of texts to embed
- `outputField` - Output field name (default: "embeddings")

### `aimodule.toolcall`

Tool calling node (uses chat with tools).

Same config as `aimodule.chat` with tools enabled.

## Examples

See `examples/aimodule-workflow/` for complete examples:
- `main.go` - Basic chat workflow
- `workflow-tool-calling.json` - Tool calling example
- `workflow-embedding.json` - Embedding example

## Architecture

```
pkg/aimodule/
├── types.go          - Core types and interfaces
├── client.go          - Client factory and registry
├── openai.go          - OpenAI implementation
├── cache.go           - Response caching
├── ratelimit.go       - Rate limiting
├── nodes_ai.go        - Workflow node handlers
├── config.go          - Configuration management
└── nodes.go           - Node registration (deprecated, handled in workflow)
```

## Future Enhancements

- Ollama local support
- Anthropic client implementation
- Vector search integration
- Redis cache backend
- Metrics and observability
- Streaming support

