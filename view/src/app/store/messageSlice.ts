import {createSlice, PayloadAction} from '@reduxjs/toolkit';
import {WsCommandType, WsReply} from "@app/model/wsCommand";

// Initial state
interface MessageState {
  lastMessage: WsReply;
}

const initialState: MessageState = {
  lastMessage: {
    type: WsCommandType.START,
    ts: 0
  },
};

// Slice for managing the server state
const messageSlice = createSlice({
  name: 'message',
  initialState,
  reducers: {
    setMessage: (state, action: PayloadAction<WsReply>) => {
      state.lastMessage = action.payload;
    },
  },
});

export const {setMessage} = messageSlice.actions;

export default messageSlice.reducer;
