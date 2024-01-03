export type OpPlayers = OpPlayer[]

export interface OpPlayer {
  uuid: string
  name: string
  level: number
  bypassesPlayerLimit: boolean
}
