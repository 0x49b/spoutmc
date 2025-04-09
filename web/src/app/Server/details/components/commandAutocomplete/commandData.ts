export const commandData = [
  {
    gamerule: [
      ['doDaylightCycle', 'naturalRegenration', 'mobGriefing', 'allowFireTicksAwayFromPlayer'],
      '<value: bool|int>'
    ]
  },
  {
    deop: ['<target: string>']
  },
  {
    op: ['<target: string>']
  },
  {
    give: [
      '<target: string>',
      {
        '<item: string>': ['stone', 'brick', 'bucket']
      }
    ]
  },
  {
    tp: [
      ['<position: x y z>']
    ]
  }
];
