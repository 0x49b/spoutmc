import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { Subscription } from '@app/model/wsCommand';

// Initial state
interface SubscriptionState {
  // todo make here a map of string<containerid or nil> and Subscription
  activeSubscriptions: Subscription[];
}

const initialState: SubscriptionState = {
  activeSubscriptions: []
};

// Slice for managing the server state
const subscriptionSlice = createSlice({
  name: 'message',
  initialState,
  reducers: {
    setSubscription: (state, action: PayloadAction<Subscription>) => {
      state.activeSubscriptions.push(action.payload);
    },
    removeSubscription: (state, action: PayloadAction<Subscription>) => {
      state.activeSubscriptions.forEach((item, index) => {
        if (item === action.payload) state.activeSubscriptions.splice(index, 1);
      });
    }
  }
});

export const { setSubscription, removeSubscription } = subscriptionSlice.actions;

export default subscriptionSlice.reducer;
