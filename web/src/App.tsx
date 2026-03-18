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
import LoginPage from './components/Auth/LoginPage';
import UserProfile from './components/Configuration/Users/UserProfile';
import UsersList from './components/Configuration/Users/UsersList';
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
    DatabaseIcon,
    ServerIcon,
    UserIcon,
    UsersIcon
} from '@patternfly/react-icons';
import smlogo from "./assets/logo.svg";

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

    const navItems = [
        {to: '/', label: 'Dashboard', icon: <ChartLineIcon/>},
        {to: '/servers', label: 'Servers', icon: <ServerIcon/>},
        {to: '/infrastructure', label: 'Infrastructure', icon: <DatabaseIcon/>},
        {to: '/plugins', label: 'Plugins', icon: <CubeIcon/>}
    ];

    // Add configuration to nav if user is admin
    if (isAdmin) {
        navItems.push({to: '/configuration', label: 'Configuration', icon: <CogIcon/>});
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
                                    toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                                        <MenuToggle
                                            ref={toggleRef}
                                            onClick={() => setIsUserMenuOpen(!isUserMenuOpen)}
                                            isExpanded={isUserMenuOpen}
                                            icon={<UserIcon/>}
                                            variant="plain"
                                        >
                                            <UserIcon/>
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
                    <NavItem
                        key={item.to}
                        itemId={item.to}
                        isActive={isActive(item.to)}
                        onClick={() => navigate(item.to)}
                    >
                        {item.icon && <span style={{marginRight: '8px'}}>{item.icon}</span>}
                        {item.label}
                    </NavItem>
                ))}
                <NavExpandable
                    title={
                        <span>
                            <span style={{marginRight: '8px'}}><UsersIcon/></span>
                            Players
                        </span>
                    }
                    itemId="players"
                    isExpanded={location.pathname.startsWith('/players')}
                    isActive={location.pathname.startsWith('/players')}
                >
                    <NavItem
                        itemId="/players/list"
                        isActive={location.pathname === '/players'}
                        onClick={() => navigate('/players')}
                    >
                        Overview
                    </NavItem>
                    <NavItem
                        itemId="/players/banned"
                        isActive={location.pathname.startsWith('/players/banned')}
                        onClick={() => navigate('/players/banned')}
                    >
                        Banned Players
                    </NavItem>
                </NavExpandable>
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
