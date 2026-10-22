package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/repository"
	"chengzi-agent-code-server/pkg/strutil"
)

const defaultInputContextLimit int64 = 200000

// LlmModelInput 大模型创建/更新入参。
type LlmModelInput struct {
	ProviderID         uint64
	Name               string
	Attachment         bool
	Reasoning          bool
	ReasoningOptions   []string
	Temperature        bool
	InputCapabilities  []string
	InputContextLimit  *int64
	OutputContextLimit *int64
	InputPrice         float64
	OutputPrice        float64
	CachePrice         float64
}

func BuildLlmModelEntity(ctx context.Context, providerRepo repository.LlmProviderRepository, in LlmModelInput) (*model.LlmModel, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.ProviderID == 0 {
		return nil, fmt.Errorf("providerId is required")
	}
	if in.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if _, err := providerRepo.GetByID(ctx, in.ProviderID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("llm provider not found")
		}
		return nil, err
	}

	caps, err := validateInputCapabilities(in.InputCapabilities)
	if err != nil {
		return nil, err
	}

	reasoningOptions := []string{}
	if in.Reasoning {
		reasoningOptions = in.ReasoningOptions
	}

	inputLimit := defaultInputContextLimit
	if in.InputContextLimit != nil {
		if *in.InputContextLimit < 0 {
			return nil, fmt.Errorf("inputContextLimit must be >= 0")
		}
		inputLimit = *in.InputContextLimit
	}
	outputLimit := int64(0)
	if in.OutputContextLimit != nil {
		if *in.OutputContextLimit < 0 {
			return nil, fmt.Errorf("outputContextLimit must be >= 0")
		}
		outputLimit = *in.OutputContextLimit
	}
	if in.InputPrice < 0 || in.OutputPrice < 0 || in.CachePrice < 0 {
		return nil, fmt.Errorf("prices must be >= 0")
	}

	return &model.LlmModel{
		ProviderID:         in.ProviderID,
		Name:               in.Name,
		Attachment:         in.Attachment,
		Reasoning:          in.Reasoning,
		ReasoningOptions:   strutil.JoinCSV(reasoningOptions),
		Temperature:        in.Temperature,
		InputCapabilities:  strutil.JoinCSV(caps),
		InputContextLimit:  inputLimit,
		OutputContextLimit: outputLimit,
		InputPrice:         in.InputPrice,
		OutputPrice:        in.OutputPrice,
		CachePrice:         in.CachePrice,
	}, nil
}

func validateInputCapabilities(caps []string) ([]string, error) {
	allowed := map[string]struct{}{}
	for _, opt := range model.InputCapabilityOptions() {
		allowed[opt] = struct{}{}
	}
	out := make([]string, 0, len(caps))
	seen := map[string]struct{}{}
	for _, c := range caps {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, ok := allowed[c]; !ok {
			return nil, fmt.Errorf("invalid input capability: %s", c)
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	return out, nil
}
