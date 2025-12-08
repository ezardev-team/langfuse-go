# Langfuse Go SDK


[![GoDoc](https://godoc.org/github.com/henomis/langfuse-go?status.svg)](https://godoc.org/github.com/henomis/langfuse-go) [![Go Report Card](https://goreportcard.com/badge/github.com/henomis/langfuse-go)](https://goreportcard.com/report/github.com/henomis/langfuse-go) [![GitHub release](https://img.shields.io/github/release/henomis/langfuse-go.svg)](https://github.com/henomis/langfuse-go/releases)

This is [Langfuse](https://langfuse.com)'s **unofficial** Go client, designed to enable you to use Langfuse's services easily from your own applications.

## Langfuse

[Langfuse](https://langfuse.com) traces, evals, prompt management and metrics to debug and improve your LLM application.


## API support

| **Index Operations**  | **Status** |
| --- | --- |
| Trace | 🟢 | 
| Generation | 🟢 |
| Span | 🟢 |
| Event | 🟢 |
| Score | 🟢 |
| Prompt (retrieve) | 🟢 |




## Getting started

### Installation

You can load langfuse-go into your project by using:
```
go get github.com/ezardev-team/langfuse-go
```


### Configuration
Just like the official Python SDK, these three environment variables will be used to configure the Langfuse client:

- `LANGFUSE_HOST`: The host of the Langfuse service.
- `LANGFUSE_PUBLIC_KEY`: Your public key for the Langfuse service.
- `LANGFUSE_SECRET_KEY`: Your secret key for the Langfuse service.


### Usage

Please refer to the [examples folder](examples/cmd/) to see how to use the SDK.

Here below a simple usage example:

```go
package main

import (
        "context"
        "fmt"

        "github.com/ezardev-team/langfuse-go"
        "github.com/ezardev-team/langfuse-go/model"
)

func main() {
        l := langfuse.New()

	err := l.Trace(&model.Trace{Name: "test-trace"})
	if err != nil {
		panic(err)
	}

	err = l.Span(&model.Span{Name: "test-span"})
	if err != nil {
		panic(err)
	}

	err = l.Generation(
		&model.Generation{
			Name:  "test-generation",
			Model: "gpt-3.5-turbo",
			ModelParameters: model.M{
				"maxTokens":   "1000",
				"temperature": "0.9",
			},
			Input: []model.M{
				{
					"role":    "system",
					"content": "You are a helpful assistant.",
				},
				{
					"role":    "user",
					"content": "Please generate a summary of the following documents \nThe engineering department defined the following OKR goals...\nThe marketing department defined the following OKR goals...",
				},
			},
			Metadata: model.M{
				"key": "value",
			},
		},
	)
	if err != nil {
		panic(err)
	}

	err = l.Event(
		&model.Event{
			Name: "test-event",
			Metadata: model.M{
				"key": "value",
			},
			Input: model.M{
				"key": "value",
			},
			Output: model.M{
				"key": "value",
			},
		},
	)
	if err != nil {
		panic(err)
	}

	err = l.GenerationEnd(
		&model.Generation{
			Output: model.M{
				"completion": "The Q3 OKRs contain goals for multiple teams...",
			},
		},
	)
	if err != nil {
		panic(err)
	}

	err = l.Score(
		&model.Score{
			Name:  "test-score",
			Value: 0.9,
		},
	)
	if err != nil {
		panic(err)
	}

        err = l.SpanEnd(&model.Span{})
        if err != nil {
                panic(err)
        }

        prompt, err := l.Prompt(context.Background(), "my-prompt", &model.PromptRequestOptions{Label: "latest"})
        if err != nil {
                panic(err)
        }

        fmt.Printf("Retrieved prompt version %d with label %s\n", prompt.Version, prompt.Label)

	l.Flush(context.Background())

}
```

### Reusing cached LLM outputs

If you store a cache key in `generation.metadata["cache_key"]`, you can avoid re-calling the LLM when that input repeats:

```go
ctx := context.Background()
l := langfuse.New(ctx)

cacheKey := "<function + model + temperature + normalized input hash>"
if hit, err := l.FindCachedGeneration(ctx, cacheKey, &langfuse.GenerationCacheOptions{Name: "summarize_10k_item_7"}); err != nil {
        panic(err)
} else if hit != nil {
        fmt.Printf("cache hit, returning prior output: %v\n", hit.Output)
        return
}

// otherwise call your LLM and ingest a new generation, persisting the cache key in metadata
gen, err := l.Generation(&model.Generation{
        TraceID:  traceID,
        Name:     "summarize_10k_item_7",
        Metadata: model.M{"cache_key": cacheKey},
        Input:    normalizedInput,
        Model:    "gemini-1.5-pro",
}, nil)
if err != nil {
        panic(err)
}
// ...
```

You can also fetch multiple cached generations in one API call to reduce latency:

```go
hits, err := l.FindCachedGenerationBatch(ctx, []string{"cache-key-1", "cache-key-2"}, nil)
if err != nil {
        panic(err)
}

if hit, ok := hits["cache-key-1"]; ok {
        fmt.Printf("found cached output for key 1: %v\n", hit.Output)
}

if hit, ok := hits["cache-key-2"]; ok {
        fmt.Printf("found cached output for key 2: %v\n", hit.Output)
}
```

## Who uses langfuse-go?

* [LinGoose](https://github.com/henomis/lingoose) Go framework for building awesome LLM apps
