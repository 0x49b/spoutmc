import { create } from 'zustand';

export type ToastVariant = 'success' | 'info' | 'warning' | 'danger';

export interface AppNotification {
  id: string;
  variant: ToastVariant;
  title: string;
  description?: string;
  createdAt: number;
}

export type PushToastInput = {
  variant: ToastVariant;
  title: string;
  description?: string;
};

function genId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 11)}`;
}

interface NotificationState {
  toasts: AppNotification[];
  /** Non-success toasts land here after the toast dismisses (timeout or close). Success toasts are never stored. */
  drawerItems: AppNotification[];
  pushToast: (input: PushToastInput) => void;
  /** Called when toast times out or user closes it. Success: removed only; other variants: copied to drawer. */
  dismissToast: (id: string) => void;
  removeFromDrawer: (id: string) => void;
}

export const useNotificationStore = create<NotificationState>((set, get) => ({
  toasts: [],
  drawerItems: [],

  pushToast: (input) => {
    const item: AppNotification = {
      ...input,
      id: genId(),
      createdAt: Date.now()
    };
    set((s) => ({ toasts: [item, ...s.toasts] }));
  },

  dismissToast: (id) => {
    const { toasts } = get();
    const item = toasts.find((t) => t.id === id);
    if (!item) return;
    set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) }));
    if (item.variant !== 'success') {
      set((s) => ({ drawerItems: [item, ...s.drawerItems] }));
    }
  },

  removeFromDrawer: (id) =>
    set((s) => ({ drawerItems: s.drawerItems.filter((t) => t.id !== id) }))
}));
