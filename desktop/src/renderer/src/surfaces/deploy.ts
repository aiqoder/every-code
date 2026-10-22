import { reactive } from 'vue'
import type { DeployTask } from '../../../shared/types'
import { useSessionStore } from '../stores/session'
import { asText, briefProject } from './format'
import { fail, ok, params, type PageSurface } from './types'

/** 部署页草稿。执行由用户点按钮发起，不做成工具，避免智能体自己调用自己。 */
export const deployPage = reactive({
  tasks: [] as DeployTask[],
  open: false,
  editingId: 0,
  title: '',
  prompt: ''
})

export const deploySurface: PageSurface = {
  channel: 'deploy',
  page: 'deploy',
  title: '部署',
  snapshot() {
    const project = useSessionStore().currentProject
    return {
      project: briefProject(project),
      tasks: deployPage.tasks.map((task) => ({
        id: task.id,
        title: task.title,
        prompt: task.prompt,
        createdAt: task.createdAt
      })),
      draft: { title: deployPage.title, prompt: deployPage.prompt, open: deployPage.open },
      note: '列表是已保存的部署任务。用户点「执行」后，说明会作为一条新消息发给你。主机连接在侧栏「主机」。'
    }
  },
  actions: [
    {
      name: 'set_draft',
      description: '填写部署对话框里的标题和说明。不要自己保存或执行；用户保存后任务才进列表，点执行后你会收到这段说明。',
      parameters: params({
        title: { type: 'string', description: '标题，可省略表示不改' },
        prompt: { type: 'string', description: '说明，可省略表示不改' }
      }),
      async run(args) {
        if (!useSessionStore().currentProject) return fail('先选择一个项目')
        if ('title' in args) deployPage.title = asText(args.title)
        if ('prompt' in args) deployPage.prompt = asText(args.prompt)
        deployPage.editingId = 0
        deployPage.open = true
        return ok('已写入部署草稿', '已写入部署草稿')
      }
    }
  ]
}
