export type BannedPlayers = BannedPlayer[]

export interface BannedPlayer {
  uuid: string
  name: string
  created: string
  source: string
  expires: string
  reason: string
}
