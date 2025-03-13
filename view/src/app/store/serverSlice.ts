import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { Server } from '@app/model/server';

// Initial state
interface ServerState {
  servers: Server[];
}

const initialState: ServerState = {
  servers: [],
};

// Slice for managing the server state
const serverSlice = createSlice({
  name: 'server',
  initialState,
  reducers: {
    setServers: (state, action: PayloadAction<Server[]>) => {
      state.servers = action.payload;
    },
  },
});

export const { setServers } = serverSlice.actions;

export default serverSlice.reducer;
