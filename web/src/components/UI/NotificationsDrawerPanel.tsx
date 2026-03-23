import React from 'react';
import {
  Button,
  ButtonVariant,
  DrawerPanelContent,
  NotificationDrawer,
  NotificationDrawerBody,
  NotificationDrawerHeader,
  NotificationDrawerList,
  NotificationDrawerListItem,
  NotificationDrawerListItemBody,
  NotificationDrawerListItemHeader
} from '@patternfly/react-core';
import TimesIcon from '@patternfly/react-icons/dist/esm/icons/times-icon';
import { useNotificationStore } from '../../store/notificationStore';

type Props = {
  onClose: () => void;
};

const NotificationsDrawerPanel: React.FC<Props> = ({ onClose }) => {
  const drawerItems = useNotificationStore((s) => s.drawerItems);
  const removeFromDrawer = useNotificationStore((s) => s.removeFromDrawer);

  return (
    <DrawerPanelContent widths={{ default: 'width_33' }} focusTrap={{ enabled: true }}>
      <NotificationDrawer>
        <NotificationDrawerHeader
          title="Notifications"
          {...(drawerItems.length === 0
            ? { customText: 'No saved notifications' }
            : { count: drawerItems.length, unreadText: 'notifications' })}
          onClose={onClose}
        />
        <NotificationDrawerBody>
          {drawerItems.length === 0 ? (
            <p
              className="pf-v6-u-text-align-center pf-v6-u-p-lg"
              style={{ color: 'var(--pf-v6-global--Color--200)' }}
            >
              When an info, warning, or error toast closes, it is listed here so you can read
              it again. Success messages are not saved.
            </p>
          ) : (
            <NotificationDrawerList aria-label="Notification list">
              {drawerItems.map((item) => (
                <NotificationDrawerListItem key={item.id} variant={item.variant}>
                  <NotificationDrawerListItemHeader title={item.title} variant={item.variant}>
                    <Button
                      variant={ButtonVariant.plain}
                      aria-label="Dismiss notification"
                      icon={<TimesIcon />}
                      onClick={() => removeFromDrawer(item.id)}
                    />
                  </NotificationDrawerListItemHeader>
                  <NotificationDrawerListItemBody
                    timestamp={new Date(item.createdAt).toLocaleString()}
                  >
                    {item.description ?? ''}
                  </NotificationDrawerListItemBody>
                </NotificationDrawerListItem>
              ))}
            </NotificationDrawerList>
          )}
        </NotificationDrawerBody>
      </NotificationDrawer>
    </DrawerPanelContent>
  );
};

export default NotificationsDrawerPanel;
