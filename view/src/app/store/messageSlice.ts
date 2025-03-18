import {createSlice, PayloadAction} from '@reduxjs/toolkit';
import {CommandType, Reply} from "@app/model/command";

// Initial state
interface MessageState {
  lastMessage: Reply;
}

const initialState: MessageState = {
  lastMessage: {
    type: CommandType.START,
    ts: 0
  },
};

// Slice for managing the server state
const messageSlice = createSlice({
  name: 'message',
  initialState,
  reducers: {
    setMessage: (state, action: PayloadAction<Reply>) => {
      state.lastMessage = action.payload;
    },
  },
});

export const {setMessage} = messageSlice.actions;

export default messageSlice.reducer;
