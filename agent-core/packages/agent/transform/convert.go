package transform

import "github.com/biome/agent-core/packages/agent/types"


func DefaultConvertToLLM(messages []types.AgentMessage) []types.Message {
	return types.ConvertToLLM(messages)
}