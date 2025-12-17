package component

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/fluxor-io/fluxor/pkg/types"
	"google.golang.org/genai"
)

// Gemini is a component that interacts with the Google Gemini API.
type Gemini struct {
	Base
	bus    types.Bus
	client *genai.Client
}

// Name returns the name of the component.
func (g *Gemini) Name() string {
	return "Gemini"
}

// OnStart is a lifecycle hook that is called when the component is started.
func (g *Gemini) OnStart(ctx context.Context, bus types.Bus) error {
	g.Base.OnStart(ctx, bus)
	g.bus = bus

	// Create a new Gemini client.
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: os.Getenv("GEMINI_API_KEY"),
	})
	if err != nil {
		return err
	}
	g.client = client

	// Create a mailbox for the component.
	mailbox := make(types.Mailbox, 128)

	// Subscribe to the /gemini/generate topic.
	if err := g.bus.Subscribe("/gemini/generate", g.Name(), mailbox); err != nil {
		return err
	}

	// Start a supervised goroutine to handle incoming messages.
	g.Go(func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-mailbox:
				g.handleGenerate(msg)
			}
		}
	})

	log.Println("Gemini component started")
	return nil
}

// OnStop is a lifecycle hook that is called when the component is stopped.
func (g *Gemini) OnStop(ctx context.Context) error {
	g.Base.OnStop(ctx)
	log.Println("Gemini component stopped")
	return nil
}

func (g *Gemini) handleGenerate(msg types.Message) {
	prompt, ok := msg.Payload.(string)
	if !ok {
		log.Println("Invalid payload type for /gemini/generate")
		return
	}

	resp, err := g.client.Models.GenerateContent(context.Background(), "gemini-pro", genai.Text(prompt), nil)
	if err != nil {
		log.Printf("Error generating content: %v", err)
		if msg.ReplyTo != "" {
			g.bus.Send(msg.ReplyTo, types.Message{Payload: fmt.Sprintf("Error: %v", err)})
		}
		return
	}

	if msg.ReplyTo != "" {
		formattedResponse := formatResponse(resp)
		g.bus.Send(msg.ReplyTo, types.Message{Payload: formattedResponse})
	}
}

// formatResponse extracts and formats the text from the Gemini API response.
func formatResponse(resp *genai.GenerateContentResponse) string {
	var responseText string
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				if part.Text != "" {
					responseText += part.Text
				}
			}
		}
	}
	return responseText
}
