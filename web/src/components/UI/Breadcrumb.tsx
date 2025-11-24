import React from 'react';
import { useLocation, Link } from 'react-router-dom';
import { Breadcrumb as PFBreadcrumb, BreadcrumbItem as PFBreadcrumbItem } from '@patternfly/react-core';
import { BreadcrumbItem } from '../../types';
import { useServerStore } from '../../store/serverStore';

const Breadcrumb: React.FC = () => {
  const location = useLocation();
  const { getServerById } = useServerStore();

  // Generate breadcrumb items based on current path
  const generateBreadcrumbs = (): BreadcrumbItem[] => {
    const pathnames = location.pathname.split('/').filter(x => x);

    const breadcrumbs: BreadcrumbItem[] = [
      { label: 'Home', path: '/' }
    ];

    let currentPath = '';

    pathnames.forEach((name, index) => {
      currentPath += `/${name}`;

      let label = name.charAt(0).toUpperCase() + name.slice(1);

      // Replace server IDs with server names
      if (name.match(/^[0-9a-fA-F-]+$/) && pathnames[index - 1] === 'servers') {
        const server = getServerById(name);
        if (server) {
          label = server.name;
        }
      }

      breadcrumbs.push({
        label,
        path: currentPath
      });
    });

    return breadcrumbs;
  };

  const breadcrumbs = generateBreadcrumbs();

  return (
    <PFBreadcrumb className="pf-v6-u-mb-md">
      {breadcrumbs.map((breadcrumb, index) => {
        const isLast = index === breadcrumbs.length - 1;

        return (
          <PFBreadcrumbItem
            key={breadcrumb.path}
            to={breadcrumb.path}
            component={(props: any) => <Link {...props} to={breadcrumb.path} />}
            isActive={isLast}
          >
            {breadcrumb.label}
          </PFBreadcrumbItem>
        );
      })}
    </PFBreadcrumb>
  );
};

export default Breadcrumb;
