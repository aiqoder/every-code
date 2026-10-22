import { inject, provide, reactive, ref, type InjectionKey } from 'vue'

/** 文件树节点：目录可再展开，文件可点击查看。 */
export interface FileNode {
  path: string
  name: string
  isDir: boolean
  size: number
}

/** 打开的文件内容状态。 */
export interface OpenedFile {
  path: string
  name: string
  content: string
  loading: boolean
  error: string
}

interface DirState {
  loading: boolean
  loaded: boolean
  error: string
  children: FileNode[]
}

export interface WorkspaceIO {
  listDir(path: string): Promise<FileNode[]>
  readFile(path: string): Promise<string>
}

export interface WorkspaceFiles {
  panelOpen: boolean
  root: string
  expanded: Record<string, boolean>
  dirs: Record<string, DirState>
  openedFile: OpenedFile | null
  togglePanel(): void
  closePanel(): void
  setRoot(path: string): void
  loadDir(path: string, force?: boolean): Promise<void>
  toggleDir(path: string): Promise<void>
  openFile(node: FileNode): Promise<void>
  closeFile(): void
}

/** 目录在前、同类按名称排序。 */
function sortNodes(rows: FileNode[]): FileNode[] {
  return [...rows].sort((a, b) => {
    if (a.isDir !== b.isDir) return a.isDir ? -1 : 1
    return a.name.localeCompare(b.name)
  })
}

/** 文件树状态。列目录和读文件由调用方注入，组件本身不依赖 Wails。 */
export function createWorkspaceFileState(io: WorkspaceIO) {
  const panelOpen = ref(false)
  const root = ref('')
  const expanded = ref<Record<string, boolean>>({})
  const dirs = ref<Record<string, DirState>>({})
  const openedFile = ref<OpenedFile | null>(null)

  function togglePanel() {
    panelOpen.value = !panelOpen.value
  }

  function closePanel() {
    panelOpen.value = false
  }

  function setRoot(path: string) {
    if (path === root.value) return
    root.value = path
    expanded.value = {}
    dirs.value = {}
  }

  async function loadDir(path: string, force = false) {
    const target = path || root.value
    if (!target) return
    const cur = dirs.value[target]
    if (!force && cur?.loaded) return
    dirs.value = {
      ...dirs.value,
      [target]: { loading: true, loaded: false, error: '', children: cur?.children ?? [] },
    }
    try {
      const rows = (await io.listDir(target)) ?? []
      dirs.value = {
        ...dirs.value,
        [target]: { loading: false, loaded: true, error: '', children: sortNodes(rows) },
      }
    } catch (e) {
      dirs.value = {
        ...dirs.value,
        [target]: {
          loading: false,
          loaded: false,
          error: e instanceof Error ? e.message : String(e),
          children: [],
        },
      }
    }
  }

  async function toggleDir(path: string) {
    const next = { ...expanded.value, [path]: !expanded.value[path] }
    expanded.value = next
    if (next[path]) await loadDir(path)
  }

  async function openFile(node: FileNode) {
    openedFile.value = {
      path: node.path,
      name: node.name,
      content: '',
      loading: true,
      error: '',
    }
    try {
      const text = await io.readFile(node.path)
      openedFile.value = {
        path: node.path,
        name: node.name,
        content: text ?? '',
        loading: false,
        error: '',
      }
    } catch (e) {
      openedFile.value = {
        path: node.path,
        name: node.name,
        content: '',
        loading: false,
        error: e instanceof Error ? e.message : String(e),
      }
    }
  }

  function closeFile() {
    openedFile.value = null
  }

  return {
    panelOpen,
    root,
    expanded,
    dirs,
    openedFile,
    togglePanel,
    closePanel,
    setRoot,
    loadDir,
    toggleDir,
    openFile,
    closeFile,
  }
}

/** 给网页注入用：字段在脚本和模板里都是解开后的值。 */
export function createWorkspaceFiles(io: WorkspaceIO): WorkspaceFiles {
  return reactive(createWorkspaceFileState(io)) as unknown as WorkspaceFiles
}

export const workspaceFilesKey: InjectionKey<WorkspaceFiles> = Symbol('workspaceFiles')

export function provideWorkspaceFiles(files: WorkspaceFiles) {
  provide(workspaceFilesKey, files)
}

export function useWorkspaceFiles(): WorkspaceFiles {
  const files = inject(workspaceFilesKey)
  if (!files) throw new Error('未注入工作目录文件')
  return files
}
