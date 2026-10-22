/** 这次点击落在刚划选的文字上时，不要当成打开或切换。 */
export function clickOnSelection(event: MouseEvent): boolean {
  const selection = window.getSelection()
  if (!selection || selection.isCollapsed) return false
  if (!selection.toString().trim()) return false
  const host = event.currentTarget
  const node = selection.anchorNode
  return host instanceof Node && !!node && host.contains(node)
}
