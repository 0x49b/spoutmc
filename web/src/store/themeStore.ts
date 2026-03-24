import {create} from 'zustand';
import {persist} from 'zustand/middleware';

export type Theme = 'light' | 'dark' | 'system';

const PF_DARK_THEME_CLASS = 'pf-v6-theme-dark';
const SYSTEM_DARK_MEDIA_QUERY = '(prefers-color-scheme: dark)';

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

if (typeof window !== 'undefined') {
  const darkModeMediaQuery = window.matchMedia(SYSTEM_DARK_MEDIA_QUERY);

  const applyThemeClasses = (isDark: boolean) => {
    const root = document.documentElement;
    root.classList.toggle(PF_DARK_THEME_CLASS, isDark);
    root.classList.toggle('dark', isDark);
    root.style.colorScheme = isDark ? 'dark' : 'light';
  };

  const applyTheme = () => {
    const theme = useThemeStore.getState().theme;
    const isDark = theme === 'dark' || (theme === 'system' && darkModeMediaQuery.matches);
    applyThemeClasses(isDark);
  };

  // Apply immediately and after persisted state rehydrates
  applyTheme();

  // React to explicit user theme changes
  useThemeStore.subscribe(() => applyTheme());

  // React to OS theme changes while in system mode
  const handleSystemThemeChange = () => {
    if (useThemeStore.getState().theme === 'system') {
      applyTheme();
    }
  };

  if (typeof darkModeMediaQuery.addEventListener === 'function') {
    darkModeMediaQuery.addEventListener('change', handleSystemThemeChange);
  } else {
    darkModeMediaQuery.addListener(handleSystemThemeChange);
  }
}