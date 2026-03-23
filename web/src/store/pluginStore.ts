import { create } from 'zustand';
import * as api from '../service/apiService';
import type { RegistryPluginEntry } from '../types';

interface PluginState {
  plugins: RegistryPluginEntry[];
  loading: boolean;
  error: string | null;

  fetchPlugins: () => Promise<void>;
  createPlugin: (body: {
    name: string;
    url: string;
    description?: string;
    serverNames: string[];
  }) => Promise<void>;
  updatePlugin: (
    id: string,
    body: { name: string; url: string; description?: string; serverNames: string[] }
  ) => Promise<void>;
  deletePlugin: (id: string) => Promise<void>;

  /** Registry entries that apply to this server (by Spout server name). */
  getPluginsForServer: (serverName: string) => RegistryPluginEntry[];
  getUserPluginCount: () => number;
}

export const usePluginStore = create<PluginState>((set, get) => ({
  plugins: [],
  loading: false,
  error: null,

  fetchPlugins: async () => {
    set({ loading: true, error: null });
    try {
      const { data } = await api.getPlugins();
      set({ plugins: data, loading: false });
    } catch (error) {
      set({
        error: error instanceof Error ? error.message : 'Failed to fetch plugins',
        loading: false
      });
    }
  },

  createPlugin: async (body) => {
    set({ error: null });
    await api.createRegistryPlugin(body);
    await get().fetchPlugins();
  },

  updatePlugin: async (id, body) => {
    set({ error: null });
    await api.updateRegistryPlugin(id, body);
    await get().fetchPlugins();
  },

  deletePlugin: async (id) => {
    set({ error: null });
    await api.deleteRegistryPlugin(id);
    await get().fetchPlugins();
  },

  getPluginsForServer: (serverName: string) =>
    get().plugins.filter((p) => p.serverNames.includes(serverName)),

  getUserPluginCount: () => get().plugins.filter((p) => !p.systemManaged).length
}));
