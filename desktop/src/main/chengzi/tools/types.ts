/** 一次工具调用的结果。error 为空表示成功。 */
export interface ToolResult {
  content: string
  error: string
}

/** 一次工具调用的上下文。workspace 是已解析的工作区根目录。 */
export interface ToolContext {
  workspace: string
}

/**
 * 客户端工具。形态对齐 pi 的 ToolDefinition：
 * 声明（name / description / parameters）与 execute 放在同一个对象里。
 */
export interface ClientTool {
  name: string
  label: string
  description: string
  parameters: Record<string, unknown>
  execute(args: Record<string, any>, ctx: ToolContext): Promise<ToolResult>
}

/** 上报给服务端 catalog 的工具声明，不含执行函数。 */
export interface ToolCatalogEntry {
  name: string
  description: string
  parameters: Record<string, unknown>
}
