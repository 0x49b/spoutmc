import {configureStore} from '@reduxjs/toolkit';
import serverReducer from './serverSlice';

export const store = configureStore({
  reducer: {
    server: serverReducer,
  },
});

// RootState type to infer the full state of the store
export type RootState = ReturnType<typeof store.getState>;
