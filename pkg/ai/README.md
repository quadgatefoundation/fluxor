# AI Module

The AI module provides a unified interface for interacting with various AI providers (OpenAI, Anthropic, Cursor, etc.) within the Fluxor framework.

## Features

- **Multi-provider support**: OpenAI, Anthropic, Cursor, and custom providers
- **Component integration**: Seamlessly integrates with Fluxor's component system
- **EventBus integration**: Publishes events when AI component starts/stops
- **Fail-fast validation**: Validates configuration and requests before execution
- **Simple API**: Easy-to-use interface for chat completions

## Quick Start

### Using AI Component (Recommended)

```go
package main

import (
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/ai"
    "github.com/fluxorio/fluxor/pkg/fluxor"
)

func main() {
    app, _ := fluxor.NewMainVerticle("config.json")
    
    // Create and deploy AI component
    aiConfig := ai.Config{
        Provider: ai.ProviderOpenAI,
        APIKey:  "your-api-key", // or use OPENAI_API_KEY env var
        Model:   "gpt-3.5-turbo",
    }
    aiComponent := ai.NewAIComponent(aiConfig)
    
    // Deploy verticle that uses AI
    app.DeployVerticle(NewMyVerticle(aiComponent))
    app.Start()
}

type MyVerticle struct {
    *core.BaseVerticle
    aiComponent *ai.AIComponent
}

func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    // Use AI component
    response, err := v.aiComponent.ChatSimple(ctx, "Hello, world!")
    if err != nil {
        return err
    }
    // Process response...
    return nil
}
```

### Using AI Client Directly

```go
import (
    "context"
    "github.com/fluxorio/fluxor/pkg/ai"
)

// Create client
config := ai.Config{
    Provider: ai.ProviderOpenAI,
    APIKey:  "your-api-key",
    Model:   "gpt-3.5-turbo",
}
client, err := ai.NewClient(config)
if err != nil {
    log.Fatal(err)
}

// Simple chat
response, err := client.ChatSimple(context.Background(), "What is Fluxor?")
if err != nil {
    log.Fatal(err)
}
fmt.Println(response)

// Advanced chat with multiple messages
req := &ai.ChatRequest{
    Messages: []ai.Message{
        {Role: "system", Content: "You are a helpful assistant."},
        {Role: "user", Content: "What is Fluxor?"},
    },
    Temperature: 0.7,
    MaxTokens:    500,
}
resp, err := client.Chat(context.Background(), req)
if err != nil {
    log.Fatal(err)
}
fmt.Println(resp.Choices[0].Message.Content)
```

## Configuration

### Environment Variables

You can use environment variables instead of hardcoding API keys:

- `OPENAI_API_KEY` - For OpenAI provider
- `CURSOR_API_KEY` - For Cursor provider
- `ANTHROPIC_API_KEY` - For Anthropic provider
- `AI_API_KEY` - Generic fallback

### Config Options

```go
type Config struct {
    Provider Provider // "openai", "anthropic", "cursor", "custom"
    APIKey   string   // API key (or use env var)
    BaseURL  string   // Base URL (optional, provider-specific defaults)
    Model    string   // Default model
    Timeout  string   // Request timeout (default: 60s)
}
```

## Supported Providers

### OpenAI

```go
config := ai.Config{
    Provider: ai.ProviderOpenAI,
    APIKey:  "sk-...", // or use OPENAI_API_KEY env var
    Model:   "gpt-3.5-turbo", // or "gpt-4", "o1-preview", etc.
}
```

### Anthropic

```go
config := ai.Config{
    Provider: ai.ProviderAnthropic,
    APIKey:  "sk-ant-...", // or use ANTHROPIC_API_KEY env var
    Model:   "claude-3-sonnet-20240229",
}
```

### Cursor

```go
config := ai.Config{
    Provider: ai.ProviderCursor,
    APIKey:  "cursor-...", // or use CURSOR_API_KEY env var
    Model:   "gpt-4",
}
```

### Custom Provider

```go
config := ai.Config{
    Provider: ai.ProviderCustom,
    APIKey:  "your-key",
    BaseURL: "https://your-custom-api.com/v1",
    Model:   "your-model",
}
```

## Integration with EventBus

The AI component automatically publishes events:

- `ai.ready` - Published when AI component starts successfully
- `ai.stopped` - Published when AI component stops

You can subscribe to these events:

```go
eventBus.Consumer("ai.ready").Handler(func(ctx core.FluxorContext, msg core.Message) error {
    var data map[string]interface{}
    msg.DecodeBody(&data)
    // Handle AI ready event
    return nil
})
```

## Error Handling

The AI module uses fail-fast validation:

- Invalid configuration → Error on component creation
- Missing API key → Error on client creation
- Invalid request → Error before API call
- API errors → Detailed error messages

## Examples

See `examples/` directory for complete examples using the AI module.

