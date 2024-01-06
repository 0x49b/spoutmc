export interface MCServer {
  Id: string
  Names: string[]
  Image: string
  ImageID: string
  Command: string
  Created: number
  Ports: Port[]
  Labels: Labels
  State: string
  Status: string
  HostConfig: HostConfig
  NetworkSettings: NetworkSettings
  Mounts: Mount[]
}

export interface Port {
  IP?: string
  PrivatePort: number
  PublicPort?: number
  Type: string
}

export interface GenericLabel {
  [key: string]: string | undefined;
}

export interface Labels extends GenericLabel {
  "io.spout.network": string
  "io.spout.servername": string
}

export interface HostConfig {
  NetworkMode: string
}

export interface NetworkSettings {
  Networks: Networks
}

export interface Networks {
  spoutnetwork: Spoutnetwork
}

export interface Spoutnetwork {
  IPAMConfig: any
  Links: any
  Aliases: any
  NetworkID: string
  EndpointID: string
  Gateway: string
  IPAddress: string
  IPPrefixLen: number
  IPv6Gateway: string
  GlobalIPv6Address: string
  GlobalIPv6PrefixLen: number
  MacAddress: string
  DriverOpts: any
}

export interface Mount {
  Type: string
  Source: string
  Destination: string
  Mode: string
  RW: boolean
  Propagation: string
  Name?: string
  Driver?: string
}
