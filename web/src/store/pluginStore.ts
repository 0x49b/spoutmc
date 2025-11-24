import { create } from 'zustand';
import { Plugin } from '../types';

interface PluginState {
  plugins: Plugin[];
  loading: boolean;
  error: string | null;
  
  // Actions
  fetchPlugins: () => Promise<void>;
  togglePluginStatus: (pluginId: string) => Promise<void>;
  savePluginConfig: (pluginId: string, config: Record<string, any>) => Promise<void>;
  addPlugin: (pluginData: Omit<Plugin, 'id' | 'status' | 'installedAt'>) => Promise<void>;
  
  // Selectors
  getPluginById: (id: string) => Plugin | undefined;
  getEnabledPlugins: () => Plugin[];
  getDisabledPlugins: () => Plugin[];
}

// Mock data for initial state
const mockPlugins: Plugin[] = [
  {
    id: '1',
    name: 'EssentialsX',
    version: '2.19.4',
    author: 'EssentialsX Team',
    status: 'enabled',
    description: 'Essential server features for Minecraft servers',
    dependencies: [],
    downloadUrl: 'https://example.com/essentialsx.jar',
    installedAt: new Date().toISOString()
  },
  {
    id: '2',
    name: 'WorldEdit',
    version: '7.2.10',
    author: 'EngineHub',
    status: 'enabled',
    description: 'In-game map editor for Minecraft',
    dependencies: [],
    downloadUrl: 'https://example.com/worldedit.jar',
    installedAt: new Date().toISOString()
  },
  {
    id: '3',
    name: 'WorldGuard',
    version: '7.0.7',
    author: 'EngineHub',
    status: 'enabled',
    description: 'World protection plugin',
    dependencies: ['WorldEdit'],
    downloadUrl: 'https://example.com/worldguard.jar',
    installedAt: new Date().toISOString()
  },
  {
    id: '4',
    name: 'Vault',
    version: '1.7.3',
    author: 'Milkbowl',
    status: 'enabled',
    description: 'Permissions, chat, and economy API',
    dependencies: [],
    downloadUrl: 'https://example.com/vault.jar',
    installedAt: new Date().toISOString()
  },
  {
    id: '5',
    name: 'LuckPerms',
    version: '5.4.40',
    author: 'Luck',
    status: 'enabled',
    description: 'Permission management system',
    dependencies: ['Vault'],
    downloadUrl: 'https://example.com/luckperms.jar',
    installedAt: new Date().toISOString()
  },
  {
    id: '6',
    name: 'CustomPlugin',
    version: '1.0.0',
    author: 'Admin',
    status: 'disabled',
    description: 'Custom server plugin in development',
    dependencies: ['Vault'],
    downloadUrl: 'https://example.com/custom.jar',
    installedAt: new Date().toISOString()
  }
];

export const usePluginStore = create<PluginState>((set, get) => ({
  plugins: mockPlugins,
  loading: false,
  error: null,
  
  fetchPlugins: async () => {
    set({ loading: true, error: null });
    
    try {
      // In a real app, this would be an API call
      // For demo purposes, we'll simulate a delay and return mock data
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      set({ plugins: mockPlugins, loading: false });
    } catch (error) {
      set({ 
        error: error instanceof Error ? error.message : 'Failed to fetch plugins', 
        loading: false 
      });
    }
  },
  
  togglePluginStatus: async (pluginId: string) => {
    set({ loading: true, error: null });
    
    try {
      // In a real app, this would be an API call
      // For demo purposes, we'll just update our local state
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      set(state => ({
        plugins: state.plugins.map(plugin => 
          plugin.id === pluginId
            ? { 
                ...plugin, 
                status: plugin.status === 'enabled' ? 'disabled' : 'enabled' 
              }
            : plugin
        ),
        loading: false
      }));
    } catch (error) {
      set({ 
        error: error instanceof Error ? error.message : 'Failed to toggle plugin status', 
        loading: false 
      });
    }
  },
  
  savePluginConfig: async (pluginId: string, config: Record<string, any>) => {
    set({ loading: true, error: null });
    
    try {
      // In a real app, this would be an API call
      // For demo purposes, we'll simulate a delay
      await new Promise(resolve => setTimeout(resolve, 1000));
      
      // Update plugin status if it changed in the config
      if (config.enabled !== undefined) {
        set(state => ({
          plugins: state.plugins.map(plugin => 
            plugin.id === pluginId
              ? { 
                  ...plugin, 
                  status: config.enabled ? 'enabled' : 'disabled' 
                }
              : plugin
          )
        }));
      }
      
      set({ loading: false });
    } catch (error) {
      set({ 
        error: error instanceof Error ? error.message : 'Failed to save plugin configuration', 
        loading: false 
      });
      throw error; // Re-throw to handle in the UI
    }
  },
  
  addPlugin: async (pluginData) => {
    set({ loading: true, error: null });
    
    try {
      // In a real app, this would be an API call to download and install the plugin
      // For demo purposes, we'll simulate a delay
      await new Promise(resolve => setTimeout(resolve, 2000));
      
      const newPlugin: Plugin = {
        ...pluginData,
        id: Math.random().toString(36).substr(2, 9),
        status: 'disabled',
        installedAt: new Date().toISOString()
      };
      
      set(state => ({
        plugins: [...state.plugins, newPlugin],
        loading: false
      }));
    } catch (error) {
      set({ 
        error: error instanceof Error ? error.message : 'Failed to add plugin', 
        loading: false 
      });
      throw error;
    }
  },
  
  // Selectors
  getPluginById: (id: string) => {
    return get().plugins.find(plugin => plugin.id === id);
  },
  
  getEnabledPlugins: () => {
    return get().plugins.filter(plugin => plugin.status === 'enabled');
  },
  
  getDisabledPlugins: () => {
    return get().plugins.filter(plugin => plugin.status === 'disabled');
  }
}));