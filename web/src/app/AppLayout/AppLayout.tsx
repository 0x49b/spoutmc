import * as React from 'react';
import {NavLink, useLocation} from 'react-router-dom';
import {
  Brand,
  Button,
  Masthead,
  MastheadBrand,
  MastheadContent,
  MastheadLogo,
  MastheadMain,
  MastheadToggle,
  Nav,
  NavExpandable,
  NavItem,
  NavList,
  Page,
  PageSidebar,
  PageSidebarBody,
  SkipToContent,
} from '@patternfly/react-core';
import {IAppRoute, IAppRouteGroup, routes} from '@app/routes';
import {BarsIcon} from '@patternfly/react-icons';
import spoutmclogo from '../bgimages/Logo.svg'
import SpoutToolbar from "@app/AppLayout/SpoutToolbar";


interface IAppLayout {
  children: React.ReactNode;
}

const AppLayout: React.FunctionComponent<IAppLayout> = ({children}) => {
  const [sidebarOpen, setSidebarOpen] = React.useState(true);


  const masthead = (
    <Masthead>
      <MastheadMain>
        <MastheadToggle>
          <Button
            icon={<BarsIcon/>}
            variant="plain"
            onClick={() => setSidebarOpen(!sidebarOpen)}
            aria-label="Global navigation"
          />
        </MastheadToggle>
        <MastheadBrand data-codemods>
          <MastheadLogo data-codemods>
            <Brand src={spoutmclogo} alt="SpoutMC" heights={{default: '30px'}}/>
          </MastheadLogo>
        </MastheadBrand>
      </MastheadMain>
      <MastheadContent><SpoutToolbar/></MastheadContent>
    </Masthead>
  );

  const location = useLocation();


  const renderNavItem = (route: IAppRoute, index: number) => {
    const isActive = location.pathname === route.path || (route.path !== '/' && location.pathname.startsWith(route.path));
    return (
      <NavItem key={`${route.label}-${index}`} id={`${route.label}-${index}`}
               isActive={isActive}>
        <NavLink to={route.path}>
          {route.label}
        </NavLink>
      </NavItem>
    )
  };

  const renderNavGroup = (group: IAppRouteGroup, groupIndex: number) => (
    <NavExpandable
      key={`${group.label}-${groupIndex}`}
      id={`${group.label}-${groupIndex}`}
      title={group.label}
      isActive={group.routes.some((route) => route.path === location.pathname)}
    >
      {group.routes.map((route, idx) => route.label && renderNavItem(route, idx))}
    </NavExpandable>
  );

  const Navigation = (
    <Nav id="nav-primary-simple">
      <NavList id="nav-list-simple">
        {routes.map(
          (route, idx) => route.label && (!route.routes ? renderNavItem(route, idx) : renderNavGroup(route, idx)),
        )}
      </NavList>
    </Nav>
  );

  const Sidebar = (
    <PageSidebar>
      <PageSidebarBody>{Navigation}</PageSidebarBody>
    </PageSidebar>
  );

  const pageId = 'primary-app-container';

  const PageSkipToContent = (
    <SkipToContent
      onClick={(event) => {
        event.preventDefault();
        const primaryContentContainer = document.getElementById(pageId);
        primaryContentContainer?.focus();
      }}
      href={`#${pageId}`}
    >
      Skip to Content
    </SkipToContent>
  );
  return (
    <Page
      mainContainerId={pageId}
      masthead={masthead}
      sidebar={sidebarOpen && Sidebar}
      skipToContent={PageSkipToContent}
    >
      {children}
    </Page>
  );
};

export {AppLayout};
