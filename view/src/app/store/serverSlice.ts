import {createSlice, PayloadAction} from '@reduxjs/toolkit';
import {Server} from '@app/model/server';

// Initial state
interface ServerState {
  servers: Server[];
  server: Server | undefined;
}

const initialState: ServerState = {
  servers: [],
  server: undefined
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
    }
  },
});

export const {setServers, setServer} = serverSlice.actions;

export default serverSlice.reducer;
