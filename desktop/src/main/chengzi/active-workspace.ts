let dir = ''

/** 当前项目目录。设置页不再保存工作区，工具和 catalog 都读这里。 */
export function currentWorkspace(): string {
  return dir
}

export function setCurrentWorkspace(next: string): void {
  dir = next.trim()
}
