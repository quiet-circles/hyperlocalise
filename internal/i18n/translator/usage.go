package translator

import "context"

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type usageCollectorKey struct{}

func WithUsageCollector(ctx context.Context, usage *Usage) context.Context {
	if usage == nil {
		return ctx
	}
	return context.WithValue(ctx, usageCollectorKey{}, usage)
}

func SetUsage(ctx context.Context, usage Usage) {
	collector, ok := ctx.Value(usageCollectorKey{}).(*Usage)
	if !ok || collector == nil {
		return
	}
	*collector = usage
}
