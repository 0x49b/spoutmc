import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { ReadyState } from 'react-use-websocket';

// Initial state
interface SocketState {
  readyState: ReadyState;
  readyStateString: string;
  isConnected: boolean;
}

const initialState: SocketState = {
  readyState: ReadyState.CLOSED,
  readyStateString: 'Closed',
  isConnected: false
};

// Slice for managing the server state
const socketSlice = createSlice({
  name: 'socket',
  initialState,
  reducers: {
    setSocketState: (state, action: PayloadAction<any>) => {
      state.readyState = action.payload.readyState;
      state.readyStateString = action.payload.readyStateString;
    },
    setIsConnectedStore: (state, action: PayloadAction<any>) => {
      state.isConnected = action.payload;
    }
  }
});

export const { setSocketState, setIsConnectedStore } = socketSlice.actions;

export default socketSlice.reducer;
