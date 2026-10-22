package model

// 四个工位通道。人设在 server/agents/<通道>/AGENT.md，不入库。
const (
	ChannelCreate  = "create"
	ChannelExecute = "execute"
	ChannelTest    = "test"
	ChannelDeploy  = "deploy"
)

// StationChannels 固定顺序，后台与客户端都按这个顺序展示。
func StationChannels() []string {
	return []string{ChannelCreate, ChannelExecute, ChannelTest, ChannelDeploy}
}
