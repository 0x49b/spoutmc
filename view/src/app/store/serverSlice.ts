import {createSlice, PayloadAction} from '@reduxjs/toolkit';
import {Server} from '@app/model/server';
import {ServerStats} from "@app/model/serverstats";

// Initial state
interface ServerState {
  servers: Server[];
  server: Server | undefined;
  serversStats: ServerStats[];
  serverStats: ServerStats | undefined;
}

const initialState: ServerState = {
  servers: [],
  server: undefined,
  serversStats: [],
  serverStats: undefined
};

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
    }
  },
});

export const {setServers, setServer, setServerStats, setServersStats} = serverSlice.actions;

export default serverSlice.reducer;
