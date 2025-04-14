import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { Server } from '@app/model/server';
import { ServerStats } from '@app/model/serverstats';
import { WsReply } from '@app/model/wsCommand';
import stripAnsi from 'strip-ansi';

// Initial state
interface ServerState {
  servers: Server[];
  server: Server | undefined;
  serversStats: ServerStats[];
  serverStats: ServerStats | undefined;
  serverLogs: { [key: string]: string[] };
}

const initialState: ServerState = {
  servers: [],
  server: undefined,
  serversStats: [],
  serverStats: undefined,
  serverLogs: {}
};

//\u001b[0;39m
const stripWeird = (input: string): string => input.replace(/>.{4}\r/g, '');


// Slice for managing the server state
const serverSlice = createSlice({
  name: 'server',
  initialState,
  reducers: {
    setServers: (state, action: PayloadAction<Server[]>) => {
      state.servers = action.payload;
    },
    setServer: (state, action: PayloadAction<Server>) => {
      state.server = action.payload;
    },
    setServerStats: (state, action: PayloadAction<ServerStats | undefined>) => {
      state.serverStats = action.payload;
    },
    //Todo not yet implemented on Serverside
    setServersStats: (state, action: PayloadAction<ServerStats[]>) => {
      state.serversStats = action.payload;
    },
    setServersLogs: (state, action: PayloadAction<WsReply>) => {


      if (action.payload.containerId) {

        const cid = action.payload.containerId;
        const newData = [];

        // First: in logs we have a pattern: >....\r this needs to be removed

        if (Array.isArray(action.payload.data)) {

          // @ts-ignore
          (action.payload.data as string[]).forEach((item) => {

            item = stripAnsi(item);
            item = stripWeird(item);

            // @ts-ignore
            newData.push(item);
          });
        }

        if (cid in state.serverLogs) {
          // @ts-ignore
          state.serverLogs[cid] = [...state.serverLogs[cid], ...newData];
        } else {
          // @ts-ignore
          state.serverLogs[cid] = [...newData];
        }

      }
    }
  }
});

export const {
  setServers,
  setServer,
  setServerStats,
  setServersStats,
  setServersLogs
} = serverSlice.actions;

export default serverSlice.reducer;
