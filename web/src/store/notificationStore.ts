import {create} from 'zustand';
import * as api from '../service/apiService';

export type ToastVariant = 'success' | 'info' | 'warning' | 'danger';

export interface AppNotification {
  id: string;
  variant: ToastVariant;
  title: string;
  description?: string;
  createdAt: number;
}

export interface GlobalNotification {
  id: number;
  key: string;
  severity: string;
  title: string;
  message: string;
  source: string;
  isOpen: boolean;
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
  globalItems: GlobalNotification[];
  consecutiveFailures: number;
  pushToast: (input: PushToastInput) => void;
  /** Called when toast times out or user closes it. Success: removed only; other variants: copied to drawer. */
  dismissToast: (id: string) => void;
  removeFromDrawer: (id: string) => void;
  fetchGlobalNotifications: () => Promise<void>;
  dismissGlobalNotification: (id: number) => Promise<void>;
  getBackoffDelay: () => number;
}

export const useNotificationStore = create<NotificationState>((set, get) => ({
  toasts: [],
  drawerItems: [],
  globalItems: [],
  consecutiveFailures: 0,

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
    set((s) => ({ drawerItems: s.drawerItems.filter((t) => t.id !== id) })),

  fetchGlobalNotifications: async () => {
    try {
      const response = await api.getNotifications();
      const items: GlobalNotification[] = response.data.map((n) => ({
        id: n.id,
        key: n.key,
        severity: n.severity,
        title: n.title,
        message: n.message,
        source: n.source,
        isOpen: n.isOpen,
        createdAt: new Date(n.createdAt).getTime()
      }));
      set({ globalItems: items, consecutiveFailures: 0 });
    } catch (error) {
      set((s) => ({ consecutiveFailures: s.consecutiveFailures + 1 }));
      console.error('Failed to load notifications:', error);
    }
  },

  dismissGlobalNotification: async (id) => {
    try {
      await api.dismissNotification(id);
      set((s) => ({ globalItems: s.globalItems.filter((n) => n.id !== id) }));
    } catch (error) {
      console.error('Failed to dismiss notification:', error);
    }
  },

  getBackoffDelay: () => {
    const failures = get().consecutiveFailures;
    if (failures === 0) return 10000;
    return Math.min(10000 * Math.pow(2, failures), 60000);
  }
}));
