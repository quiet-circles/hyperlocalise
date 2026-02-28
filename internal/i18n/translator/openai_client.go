package translator

import (
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

func newOpenAIClient(opts ...option.RequestOption) openai.Client {
	return openai.NewClient(opts...)
}
