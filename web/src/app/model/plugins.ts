export interface PluginServerList {
  name: string
  id: string
  plugins: Plugin[]
}

export interface Plugin {
  modifiedtime: string
  islink: boolean
  isdir: boolean
  linksto: string
  size: number
  name: string
  path: string
  children: Plugin[]
}
