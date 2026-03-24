import React, {useEffect, useMemo, useState} from 'react';
import {
    createBrowserRouter,
    Navigate,
    Outlet,
    RouterProvider,
    useLocation,
    useNavigate
} from 'react-router-dom';
import Dashboard from './components/Dashboard/Dashboard';
import ServersList from './components/Servers/ServersList';
import ServerDetail from './components/Servers/ServerDetail';
import PlayersList from './components/Players/PlayersList';
import BannedPlayersList from './components/Players/BannedPlayersList';
import PluginsList from './components/Plugins/PluginsList';
import InfrastructureList from './components/Infrastructure/InfrastructureList';
import InfrastructureDetail from './components/Infrastructure/InfrastructureDetail';
import LoginPage from './components/Auth/LoginPage';
import UserProfile from './components/Configuration/Users/UserProfile';
import UsersList from './components/Configuration/Users/UsersList';
import RolesList from './components/Configuration/Roles/RolesList';
import PermissionsAdmin from './components/Configuration/Permissions/PermissionsAdmin';
import ProtectedRoute from './components/Auth/ProtectedRoute';
import SetupWizard from './components/Setup/SetupWizard';
import {getUserAvatarDataUrl, useAuthStore} from './store/authStore';
import ThemeToggle from './components/UI/ThemeToggle';
import ToastHost from './components/UI/ToastHost';
import NotificationsDrawerPanel from './components/UI/NotificationsDrawerPanel';
import {useNotificationStore} from './store/notificationStore';

import {
    Avatar,
    Brand,
    Drawer,
    DrawerContent,
    Dropdown,
    DropdownItem,
    DropdownList,
    Masthead,
    MastheadBrand,
    MastheadContent,
    MastheadLogo,
    MastheadMain,
    MastheadToggle,
    MenuToggle,
    MenuToggleElement,
    Nav,
    NavExpandable,
    NavItem,
    NavList,
    NotificationBadge,
    NotificationBadgeVariant,
    Page,
    PageSidebar,
    PageSidebarBody,
    PageToggleButton,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem
} from '@patternfly/react-core';
import {
    BarsIcon,
    BellIcon,
    ChartLineIcon,
    CogIcon,
    CubeIcon,
    ServerIcon,
    UserIcon,
    UsersIcon
} from '@patternfly/react-icons';
import smlogo from "./assets/logo.svg";

/** Optional gates aligned with {@link ProtectedRoute}: all permission keys, and/or at least one role. */
export type NavAccess = {
    /** User must have every key (same semantics as `ProtectedRoute` `requiredPermissionKeys`). */
    requiredPermissionKeys?: string[];
    /** User must have at least one of these role names (OR). */
    requiredRoles?: string[];
};

export type NavChildItem = {
    to: string;
    label: string;
    exact?: boolean;
} & NavAccess;

export type NavEntry =
    | ({
          type: 'item';
          to: string;
          label: string;
          icon?: React.ReactNode;
      } & NavAccess)
    | ({
          type: 'group';
          id: string;
          label: string;
          icon?: React.ReactNode;
          children: NavChildItem[];
      } & NavAccess);

function canSeeNavAccess(
    access: NavAccess,
    hasPermission: (key: string) => boolean,
    hasRole: (roleName: string) => boolean
): boolean {
    const keys = access.requiredPermissionKeys;
    const roles = access.requiredRoles;
    const permOk = !keys?.length || keys.every((k) => hasPermission(k));
    const roleOk = !roles?.length || roles.some((r) => hasRole(r));
    return permOk && roleOk;
}

/** Drop entries the user may not see; trim groups with no visible children. */
function filterNavEntry(
    entry: NavEntry,
    hasPermission: (key: string) => boolean,
    hasRole: (roleName: string) => boolean
): NavEntry | null {
    if (entry.type === 'item') {
        return canSeeNavAccess(entry, hasPermission, hasRole) ? entry : null;
    }
    const visibleChildren = entry.children.filter((c) => canSeeNavAccess(c, hasPermission, hasRole));
    if (visibleChildren.length === 0) {
        return null;
    }
    if (!canSeeNavAccess(entry, hasPermission, hasRole)) {
        return null;
    }
    return {...entry, children: visibleChildren};
}

/** Declarative nav; visibility is derived via {@link filterNavEntry} (same rules as route `ProtectedRoute`). */
const NAV_ITEMS_RAW: NavEntry[] = [
    {type: 'item', to: '/', label: 'Dashboard', icon: <ChartLineIcon/>},
    {
        type: 'group',
        id: 'server-group',
        label: 'Servers',
        icon: <ServerIcon/>,
        children: [
            {to: '/servers', label: 'Game Server', requiredPermissionKeys: ['server.list.read']},
            {to: '/infrastructure', label: 'Infrastructure Server', requiredPermissionKeys: ['server.list.read']}
        ]
    },
    {
        type: 'group',
        id: 'players',
        label: 'Players',
        icon: <UsersIcon/>,
        children: [
            {to: '/players', label: 'Overview', exact: true, requiredPermissionKeys: ['player.list.read']},
            {to: '/players/banned', label: 'Banned Players', requiredPermissionKeys: ['player.list.read']}
        ]
    },
    {type: 'item', to: '/plugins', label: 'Plugins', icon: <CubeIcon/>, requiredPermissionKeys: ['server.list.read']},
    {
        type: 'group',
        id: 'configuration',
        label: 'Configuration',
        icon: <CogIcon/>,
        requiredRoles: ['admin'],
        children: [
            {to: '/users', label: 'User Management', requiredRoles: ['admin']},
            {to: '/configuration/roles', label: 'Role Management', requiredRoles: ['admin']},
            {
                to: '/configuration/permissions',
                label: 'Permissions Management',
                requiredRoles: ['admin']
            }
        ]
    }
];

// Layout wrapper component that includes Page structure for authenticated routes
const PageLayout = () => {
    const navigate = useNavigate();
    const location = useLocation();
    const {user, logout, hasPermission, hasRole} = useAuthStore();
    const toolbarAvatarSrc = getUserAvatarDataUrl(user);
    const drawerNotificationCount = useNotificationStore((s) => s.drawerItems.length);
    const globalNotificationCount = useNotificationStore((s) => s.globalItems.length);
    const fetchGlobalNotifications = useNotificationStore((s) => s.fetchGlobalNotifications);
    const getBackoffDelay = useNotificationStore((s) => s.getBackoffDelay);
    const [isUserMenuOpen, setIsUserMenuOpen] = useState(false);
    const [isSidebarOpen, setIsSidebarOpen] = useState(true);
    const [isNotificationsDrawerOpen, setIsNotificationsDrawerOpen] = useState(false);

    const isActive = (path: string) => {
        if (path === '/') {
            return location.pathname === '/';
        }
        return location.pathname.startsWith(path);
    };

    const handleLogout = async () => {
        try {
            await logout();
            navigate('/login');
        } catch (error) {
            console.error('Failed to logout:', error);
        }
    };

    useEffect(() => {
        let timeoutId: ReturnType<typeof setTimeout>;
        let cancelled = false;

        const poll = async () => {
            await fetchGlobalNotifications();
            if (!cancelled) {
                timeoutId = setTimeout(poll, getBackoffDelay());
            }
        };

        poll();
        return () => {
            cancelled = true;
            clearTimeout(timeoutId);
        };
    }, [fetchGlobalNotifications, getBackoffDelay]);

    const visibleNavItems = useMemo(
        () =>
            NAV_ITEMS_RAW.map((entry) => filterNavEntry(entry, hasPermission, hasRole)).filter(
                (e): e is NavEntry => e !== null
            ),
        [hasPermission, hasRole]
    );

    const masthead = (
        <Masthead>
            <MastheadMain>
                <MastheadToggle>
                    <PageToggleButton
                        variant="plain"
                        aria-label="Global navigation"
                        isSidebarOpen={isSidebarOpen}
                        onSidebarToggle={() => setIsSidebarOpen(!isSidebarOpen)}
                    >
                        <BarsIcon/>
                    </PageToggleButton>
                </MastheadToggle>
                <MastheadBrand>
                    <MastheadLogo component={(props) => <a {...props} href="/"/>}>
                        <Brand src={smlogo} alt="SpoutMC" heights={{default: '36px'}}/>
                    </MastheadLogo>
                </MastheadBrand>
            </MastheadMain>
            <MastheadContent>
                <Toolbar id="toolbar" isFullHeight isStatic>
                    <ToolbarContent>
                        <ToolbarGroup
                            variant="action-group-plain"
                            align={{default: 'alignEnd'}}
                            gap={{default: 'gapMd'}}
                        >
                            <ToolbarItem>
                                <ThemeToggle/>
                            </ToolbarItem>
                            <ToolbarItem>
                                <NotificationBadge
                                    variant={
                                        drawerNotificationCount + globalNotificationCount > 0
                                            ? NotificationBadgeVariant.unread
                                            : NotificationBadgeVariant.read
                                    }
                                    count={drawerNotificationCount + globalNotificationCount}
                                    icon={<BellIcon/>}
                                    isExpanded={isNotificationsDrawerOpen}
                                    aria-label="Notifications"
                                    onClick={() => setIsNotificationsDrawerOpen((open) => !open)}
                                />
                            </ToolbarItem>
                            <ToolbarItem>
                                <Dropdown
                                    isOpen={isUserMenuOpen}
                                    onSelect={() => setIsUserMenuOpen(false)}
                                    onOpenChange={(isOpen: boolean) => setIsUserMenuOpen(isOpen)}
                                    popperProps={{
                                        placement: 'bottom-end'
                                    }}
                                    toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                                        <MenuToggle
                                            ref={toggleRef}
                                            onClick={() => setIsUserMenuOpen(!isUserMenuOpen)}
                                            isExpanded={isUserMenuOpen}
                                            variant="plain"
                                            aria-label="User menu"
                                        >
                                            <span style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                                                {toolbarAvatarSrc ? (
                                                    <Avatar src={toolbarAvatarSrc} alt="" size="sm"/>
                                                ) : (
                                                    <UserIcon/>
                                                )}
                                                <span>
                                                    {user?.displayName?.trim() || user?.email || 'User'}
                                                    {user?.minecraftName?.trim() && (
                                                        <span style={{ color: 'var(--pf-v6-global--Color--200)' }}>
                                                            {' '}({user.minecraftName})
                                                        </span>
                                                    )}
                                                </span>
                                            </span>
                                        </MenuToggle>
                                    )}
                                    shouldFocusToggleOnSelect
                                >
                                    <DropdownList>
                                        <DropdownItem key="profile"
                                                      onClick={() => navigate('/profile')}>
                                            Your Profile
                                        </DropdownItem>
                                        <DropdownItem key="logout" onClick={handleLogout}>
                                            Sign Out
                                        </DropdownItem>
                                    </DropdownList>
                                </Dropdown>
                            </ToolbarItem>
                        </ToolbarGroup>
                    </ToolbarContent>
                </Toolbar>
            </MastheadContent>
        </Masthead>
    );

    const pageNav = (
        <Nav>
            <NavList>
                {visibleNavItems.map((item) => (
                    item.type === 'item' ? (
                        <NavItem
                            key={item.to}
                            itemId={item.to}
                            isActive={isActive(item.to)}
                            onClick={() => navigate(item.to)}
                        >
                            {item.icon && <span style={{marginRight: '8px'}}>{item.icon}</span>}
                            {item.label}
                        </NavItem>
                    ) : (
                        <NavExpandable
                            key={item.id}
                            title={
                                <span>
                                    {item.icon && <span style={{marginRight: '8px'}}>{item.icon}</span>}
                                    {item.label}
                                </span>
                            }
                            isExpanded={item.children.some((child) =>
                                child.exact
                                    ? location.pathname === child.to
                                    : location.pathname.startsWith(child.to)
                            )}
                            isActive={item.children.some((child) =>
                                child.exact
                                    ? location.pathname === child.to
                                    : location.pathname.startsWith(child.to)
                            )}
                        >
                            {item.children.map((child) => (
                                <NavItem
                                    key={child.to}
                                    itemId={child.to}
                                    isActive={child.exact ? location.pathname === child.to : location.pathname.startsWith(child.to)}
                                    onClick={() => navigate(child.to)}
                                >
                                    {child.label}
                                </NavItem>
                            ))}
                        </NavExpandable>
                    )
                ))}
            </NavList>
        </Nav>
    );

    const sidebar = (
        <PageSidebar isSidebarOpen={isSidebarOpen}>
            <PageSidebarBody>{pageNav}</PageSidebarBody>
        </PageSidebar>
    );

    return (
        <>
            <Drawer isExpanded={isNotificationsDrawerOpen} position="end">
                <DrawerContent
                    panelContent={
                        <NotificationsDrawerPanel
                            onClose={() => setIsNotificationsDrawerOpen(false)}
                        />
                    }
                >
                    <Page masthead={masthead} sidebar={sidebar} isManagedSidebar>
                        <Outlet/>
                    </Page>
                </DrawerContent>
            </Drawer>
            <ToastHost/>
        </>
    );
};

function App() {
    const {checkAuth} = useAuthStore();
    const [setupCompleted] = useState(
        localStorage.getItem('setupCompleted') === 'true'
    );

    useEffect(() => {
        checkAuth();
    }, [checkAuth]);

    // Create router with memoization based on setup status
    const router = useMemo(() => {
        if (!setupCompleted) {
            return createBrowserRouter([
                {
                    path: '*',
                    element: <SetupWizard/>
                }
            ]);
        }

        return createBrowserRouter([
            {
                path: '/login',
                element: <LoginPage/>
            },
            {
                path: '/',
                element: (
                    <ProtectedRoute>
                        <PageLayout/>
                    </ProtectedRoute>
                ),
                children: [
                    {
                        index: true,
                        element: <Dashboard/>
                    },
                    {
                        path: 'profile',
                        element: <UserProfile/>
                    },
                    {
                        path: 'configuration',
                        element: (
                            <ProtectedRoute requireAdmin>
                                <Navigate to="/users" replace/>
                            </ProtectedRoute>
                        )
                    },
                    {
                        path: 'users',
                        element: (
                            <ProtectedRoute requireAdmin>
                                <UsersList/>
                            </ProtectedRoute>
                        )
                    },
                    {
                        path: 'configuration/roles',
                        element: (
                            <ProtectedRoute requireAdmin>
                                <RolesList/>
                            </ProtectedRoute>
                        )
                    },
                    {
                        path: 'configuration/permissions',
                        element: (
                            <ProtectedRoute requireAdmin>
                                <PermissionsAdmin/>
                            </ProtectedRoute>
                        )
                    },
                    {
                        path: 'servers',
                        element: (
                            <ProtectedRoute
                                requiredPermissionKeys={['server.list.read']}>
                                <ServersList/>
                            </ProtectedRoute>
                        )
                    },
                    {
                        path: 'servers/:id',
                        element: (
                            <ProtectedRoute
                                requiredPermissionKeys={['server.list.read']}>
                                <ServerDetail/>
                            </ProtectedRoute>
                        )
                    },
                    {
                        path: 'infrastructure',
                        element: (
                            <ProtectedRoute
                                requiredPermissionKeys={['server.list.read']}>
                                <InfrastructureList/>
                            </ProtectedRoute>
                        )
                    },
                    {
                        path: 'infrastructure/:id',
                        element: (
                            <ProtectedRoute
                                requiredPermissionKeys={['server.list.read']}>
                                <InfrastructureDetail/>
                            </ProtectedRoute>
                        )
                    },
                    {
                        path: 'players',
                        element: (
                            <ProtectedRoute
                                requiredPermissionKeys={['player.list.read']}>
                                <PlayersList/>
                            </ProtectedRoute>
                        )
                    },
                    {
                        path: 'players/banned',
                        element: (
                            <ProtectedRoute
                                requiredPermissionKeys={['player.list.read']}>
                                <BannedPlayersList/>
                            </ProtectedRoute>
                        )
                    },
                    {
                        path: 'plugins',
                        element: (
                            <ProtectedRoute
                                requiredPermissionKeys={['server.list.read']}>
                                <PluginsList/>
                            </ProtectedRoute>
                        )
                    },
                    {
                        path: '*',
                        element: <Navigate to="/" replace/>
                    }
                ]
            }
        ]);
    }, [setupCompleted]);

    return <RouterProvider router={router}/>;
}

export default App;
