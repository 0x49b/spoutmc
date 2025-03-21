import * as React from 'react';
import {Navigate, Route, Routes} from 'react-router-dom';
import { ServerList } from '@app/Server/ServerList';
import { Support } from '@app/Support/Support';
import { GeneralSettings } from '@app/Settings/General/GeneralSettings';
import { ProfileSettings } from '@app/Settings/Profile/ProfileSettings';
import { NotFound } from '@app/NotFound/NotFound';
import {ServerDetail} from "@app/Server/details/ServerDetail";

export interface IAppRoute {
  label?: string; // Excluding the label will exclude the route from the nav sidebar in AppLayout
  /* eslint-disable @typescript-eslint/no-explicit-any */
  element: React.ReactElement;
  /* eslint-enable @typescript-eslint/no-explicit-any */
  exact?: boolean;
  path: string;
  title: string;
  routes?: undefined;
}

export interface IAppRouteGroup {
  label: string;
  routes: IAppRoute[];
}

export type AppRouteConfig = IAppRoute | IAppRouteGroup;

const routes: AppRouteConfig[] = [
  {
    element: <ServerList />,
    exact: true,
    label: 'Serverlist',
    path: '/server',
    title: 'Serverlist'
  },
  {
    element: <ServerDetail/>,
    exact: true,
    path: '/server/detail/:serverId',
    title: 'Server Details'
  },
  {
    element: <Support />,
    exact: true,
    label: 'Support',
    path: '/support',
    title: 'PatternFly Seed | Support Page'
  },
  {
    label: 'Settings',
    routes: [
      {
        element: <GeneralSettings />,
        exact: true,
        label: 'General',
        path: '/settings/general',
        title: 'PatternFly Seed | General Settings'
      },
      {
        element: <ProfileSettings />,
        exact: true,
        label: 'Profile',
        path: '/settings/profile',
        title: 'PatternFly Seed | Profile Settings'
      },
    ],
  },
];

const flattenedRoutes: IAppRoute[] = routes.reduce(
  (flattened, route) => [...flattened, ...(route.routes ? route.routes : [route])],
  [] as IAppRoute[],
);

const AppRoutes = (): React.ReactElement => (
  <Routes>

    <Route path="/" element={<Navigate to="/server" replace />} />

    {flattenedRoutes.map(({ path, element }, idx) => (
     <Route path={path} element={element} key={idx} />
    ))}
    <Route element={<NotFound />} />
  </Routes>
);

export { AppRoutes, routes };
