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
import Configuration from './components/Configuration/Configuration';
import ProtectedRoute from './components/Auth/ProtectedRoute';
import SetupWizard from './components/Setup/SetupWizard';
import {useAuthStore} from './store/authStore';
import ThemeToggle from './components/UI/ThemeToggle';

import {
    Brand,
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
    ChartLineIcon,
    CogIcon,
    CubeIcon,
    ServerIcon,
    UserIcon,
    UsersIcon
} from '@patternfly/react-icons';
import smlogo from "./assets/logo.svg";

type NavChildItem = {
    to: string;
    label: string;
    exact?: boolean;
};

type NavEntry =
    | {
    type: 'item';
    to: string;
    label: string;
    icon?: React.ReactNode;
}
    | {
    type: 'group';
    id: string;
    label: string;
    icon?: React.ReactNode;
    children: NavChildItem[];
};

// Layout wrapper component that includes Page structure for authenticated routes
const PageLayout = () => {
    const navigate = useNavigate();
    const location = useLocation();
    const {user, logout} = useAuthStore();
    const [isUserMenuOpen, setIsUserMenuOpen] = useState(false);
    const [isSidebarOpen, setIsSidebarOpen] = useState(true);

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

    const isAdmin = user?.roles.includes('admin');

    const navItems: NavEntry[] = [
        {type: 'item', to: '/', label: 'Dashboard', icon: <ChartLineIcon/>},
        {
            type: 'group',
            id: 'server-group',
            label: 'Servers',
            icon: <ServerIcon/>,
            children: [
                {to: '/servers', label: 'Game Server'},
                {to: '/infrastructure', label: 'Infrastructure Server'}
            ]
        },
        {
            type: 'group',
            id: 'players',
            label: 'Players',
            icon: <UsersIcon/>,
            children: [
                {to: '/players', label: 'Overview', exact: true},
                {to: '/players/banned', label: 'Banned Players'}
            ]
        },
        {type: 'item', to: '/plugins', label: 'Plugins', icon: <CubeIcon/>}
    ];

    // Add configuration to nav if user is admin
    if (isAdmin) {
        navItems.push({type: 'item', to: '/configuration', label: 'Configuration', icon: <CogIcon/>});
    }

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
                                                <UserIcon/>
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
                {navItems.map((item) => (
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
        <Page masthead={masthead} sidebar={sidebar} isManagedSidebar>
            <Outlet/>
        </Page>
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
                            <ProtectedRoute
                                requiredPermissions={[{action: 'manage', subject: 'users'}]}>
                                <Configuration/>
                            </ProtectedRoute>
                        )
                    },
                    {
                        path: 'users',
                        element: (
                            <ProtectedRoute
                                requiredPermissions={[{action: 'manage', subject: 'users'}]}>
                                <UsersList/>
                            </ProtectedRoute>
                        )
                    },
                    {
                        path: 'configuration/roles',
                        element: (
                            <ProtectedRoute
                                requiredPermissions={[{action: 'manage', subject: 'users'}]}>
                                <RolesList/>
                            </ProtectedRoute>
                        )
                    },
                    {
                        path: 'servers',
                        element: (
                            <ProtectedRoute
                                requiredPermissions={[{action: 'read', subject: 'servers'}]}>
                                <ServersList/>
                            </ProtectedRoute>
                        )
                    },
                    {
                        path: 'servers/:id',
                        element: (
                            <ProtectedRoute
                                requiredPermissions={[{action: 'read', subject: 'servers'}]}>
                                <ServerDetail/>
                            </ProtectedRoute>
                        )
                    },
                    {
                        path: 'infrastructure',
                        element: (
                            <ProtectedRoute
                                requiredPermissions={[{action: 'read', subject: 'servers'}]}>
                                <InfrastructureList/>
                            </ProtectedRoute>
                        )
                    },
                    {
                        path: 'infrastructure/:id',
                        element: (
                            <ProtectedRoute
                                requiredPermissions={[{action: 'read', subject: 'servers'}]}>
                                <InfrastructureDetail/>
                            </ProtectedRoute>
                        )
                    },
                    {
                        path: 'players',
                        element: (
                            <ProtectedRoute
                                requiredPermissions={[{action: 'read', subject: 'players'}]}>
                                <PlayersList/>
                            </ProtectedRoute>
                        )
                    },
                    {
                        path: 'players/banned',
                        element: (
                            <ProtectedRoute
                                requiredPermissions={[{action: 'read', subject: 'players'}]}>
                                <BannedPlayersList/>
                            </ProtectedRoute>
                        )
                    },
                    {
                        path: 'plugins',
                        element: (
                            <ProtectedRoute
                                requiredPermissions={[{action: 'read', subject: 'servers'}]}>
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
