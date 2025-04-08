export const commandData = [
  {
    gamerule: [
      ["doDaylightCycle", "naturalRegenration", "mobGriefing"],
      {
        "<item: bool>": ["false", "true"]
      }
    ]
  },
  {
    deop: ["<target: string>"]
  },
  {
    give: [
      "<target: string>",
      {
        "<item: string>": ["stone", "brick", "bucket"]
      }
    ]
  },
  {
    tp: [
      ['<position: x y z>']]
  }
]
