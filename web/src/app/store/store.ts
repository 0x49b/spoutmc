import { configureStore } from '@reduxjs/toolkit';
import serverReducer from './serverSlice';
import messageReducer from './messageSlice';
import socketReducer from './socketSlice';
import subscriptionReducer from './subscriptionSlice';

export const store = configureStore({
  reducer: {
    server: serverReducer,
    message: messageReducer,
    socket: socketReducer,
    activeSubscriptions: subscriptionReducer
  }
});

// RootState type to infer the full state of the store
export type RootState = ReturnType<typeof store.getState>;
