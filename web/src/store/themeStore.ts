import { create } from 'zustand';
import { persist } from 'zustand/middleware';

type Theme = 'light' | 'dark' | 'system';

interface ThemeState {
  theme: Theme;
  setTheme: (theme: Theme) => void;
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set) => ({
      theme: 'system',
      setTheme: (theme) => set({ theme })
    }),
    {
      name: 'theme-storage'
    }
  )
);

// Apply theme on store initialization and system theme changes
if (typeof window !== 'undefined') {
  const darkModeMediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
  
  const applyTheme = () => {
    const theme = useThemeStore.getState().theme;
    const isDark = theme === 'dark' || (theme === 'system' && darkModeMediaQuery.matches);
    document.documentElement.classList.toggle('dark', isDark);
  };
  
  // Apply theme initially
  applyTheme();
  
  // Subscribe to theme changes
  useThemeStore.subscribe(() => applyTheme());
  
  // Listen for system theme changes
  darkModeMediaQuery.addEventListener('change', applyTheme);
}