/**
 * 工位智能体。id、name 与 server/agents 下 AGENT.md 一致。
 * 发消息时直接使用这里的 id，不按通道再向服务端查询。
 */
export const advisorAgent = { id: 'ade-advisor', name: '规划' }
export const envAgent = { id: 'ade-envcheck', name: '环境检测' }
export const coderAgent = { id: 'ade-coder', name: '开发' }
export const verifierAgent = { id: 'ade-verifier', name: '验收' }
export const reviewerAgent = { id: 'ade-reviewer', name: '复核' }
export const deployAgent = { id: 'ade-deploy', name: '部署' }

/** 开始执行和重新执行时一并交给服务端的固定智能体。 */
export function pipelineAgents(): { coder: string; verifier: string; reviewer: string } {
  return { coder: coderAgent.id, verifier: verifierAgent.id, reviewer: reviewerAgent.id }
}
