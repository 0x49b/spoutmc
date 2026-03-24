import React from 'react';
import {
    Dropdown,
    DropdownItem,
    DropdownList,
    MenuToggle,
    MenuToggleElement
} from '@patternfly/react-core';
import {DesktopIcon, MoonIcon, SunIcon} from '@patternfly/react-icons';
import {Theme, useThemeStore} from '../../store/themeStore';

const ThemeToggle: React.FC = () => {
  const { theme, setTheme } = useThemeStore();
  const [isOpen, setIsOpen] = React.useState(false);

  const getThemeIcon = (value: Theme) => {
    switch (value) {
      case 'light':
        return <SunIcon />;
      case 'dark':
        return <MoonIcon />;
      case 'system':
      default:
        return <DesktopIcon />;
    }
  };

  const getThemeLabel = (value: Theme) => {
    switch (value) {
      case 'light':
        return 'Light';
      case 'dark':
        return 'Dark';
      case 'system':
      default:
        return 'System';
    }
  };

  return (
    <Dropdown
      isOpen={isOpen}
      onSelect={() => setIsOpen(false)}
      onOpenChange={(open: boolean) => setIsOpen(open)}
      popperProps={{
        placement: 'bottom-end'
      }}
      toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
        <MenuToggle
          ref={toggleRef}
          onClick={() => setIsOpen(!isOpen)}
          isExpanded={isOpen}
          variant="plain"
          icon={getThemeIcon(theme)}
          aria-label={`Theme menu. Current theme: ${getThemeLabel(theme)}`}
        />
      )}
      shouldFocusToggleOnSelect
    >
      <DropdownList>
        <DropdownItem
          key="theme-light"
          isActive={theme === 'light'}
          icon={<SunIcon />}
          onClick={() => setTheme('light')}
        >
          Light
        </DropdownItem>
        <DropdownItem
          key="theme-dark"
          isActive={theme === 'dark'}
          icon={<MoonIcon />}
          onClick={() => setTheme('dark')}
        >
          Dark
        </DropdownItem>
        <DropdownItem
          key="theme-system"
          isActive={theme === 'system'}
          icon={<DesktopIcon />}
          onClick={() => setTheme('system')}
        >
          System
        </DropdownItem>
      </DropdownList>
    </Dropdown>
  );
};

export default ThemeToggle;
