/** 主进程与渲染进程之间的固定通道名。两边都从这里引用，避免拼写漂移。 */
export const IPC = {
  // 连接与服务
  settingsGet: 'ade:settings:get',
  settingsSave: 'ade:settings:save',
  login: 'ade:login',
  logout: 'ade:logout',
  connect: 'ade:connect',
  statusGet: 'ade:status:get',
  statusEvent: 'ade:status:event',

  // 目录选择
  pickDirectory: 'ade:pick-directory',
  pickFile: 'ade:pick-file',

  // PTY
  ptyCreate: 'ade:pty:create',
  ptyWrite: 'ade:pty:write',
  ptyResize: 'ade:pty:resize',
  ptyClose: 'ade:pty:close',
  ptyData: 'ade:pty:data',
  ptyExit: 'ade:pty:exit',
  shellOpenInEditor: 'ade:shell:open-external',

  // Git
  gitInit: 'ade:git:init',
  gitStatus: 'ade:git:status',

  // 本机技能
  skillListLocal: 'ade:skill:list-local',
  skillOpenDir: 'ade:skill:open-dir',
  skillImport: 'ade:skill:import',

  // SSH
  sshOpen: 'ade:ssh:open',
  sshClose: 'ade:ssh:close',
  release: 'ade:release',

  // 当前页面的操作，供侧边智能体调用
  pageSurfacePublish: 'ade:page-surface:publish',
  pageSurfaceCall: 'ade:page-surface:call',
  pageSurfaceResult: 'ade:page-surface:result'
} as const
