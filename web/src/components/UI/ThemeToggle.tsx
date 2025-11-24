import React from 'react';
import { Button } from '@patternfly/react-core';
import { MoonIcon, SunIcon } from '@patternfly/react-icons';
import { useThemeStore } from '../../store/themeStore';

const ThemeToggle: React.FC = () => {
  const { theme, setTheme } = useThemeStore();

  const cycleTheme = () => {
    switch (theme) {
      case 'light':
        setTheme('dark');
        break;
      case 'dark':
        setTheme('system');
        break;
      case 'system':
        setTheme('light');
        break;
    }
  };

  const Icon = theme === 'light' ? SunIcon : MoonIcon;

  return (
    <Button
      variant="plain"
      onClick={cycleTheme}
      aria-label={`Current theme: ${theme}`}
      icon={<Icon />}
    >
      <Icon />
    </Button>
  );
};

export default ThemeToggle;
