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
import {useNotificationStore} from '../../store/notificationStore';


type Props = {
  onClose: () => void;
};

const NotificationsDrawerPanel: React.FC<Props> = ({ onClose }) => {
  const drawerItems = useNotificationStore((s) => s.drawerItems);
  const globalItems = useNotificationStore((s) => s.globalItems);
  const removeFromDrawer = useNotificationStore((s) => s.removeFromDrawer);
  const dismissGlobalNotification = useNotificationStore((s) => s.dismissGlobalNotification);

  const totalItems = globalItems.length + drawerItems.length;

  const mapVariant = (severity: string): 'info' | 'warning' | 'danger' | 'success' => {
    if (severity === 'danger') return 'danger';
    if (severity === 'warning') return 'warning';
    if (severity === 'success') return 'success';
    return 'info';
  };

  return (
    <DrawerPanelContent widths={{ default: 'width_33' }} focusTrap={{ enabled: true }}>
      <NotificationDrawer>
        <NotificationDrawerHeader
          title="Notifications"
          {...(totalItems === 0
            ? { customText: 'No saved notifications' }
            : { count: totalItems, unreadText: 'notifications' })}
          onClose={onClose}
        />
        <NotificationDrawerBody>
          {totalItems === 0 ? (
            <p
              className="pf-v6-u-text-align-center pf-v6-u-p-lg p-1"
              style={{ color: 'var(--pf-v6-global--Color--200)' }}
            >
              When an info, warning, or error toast closes, it is listed here so you can read
              it again. Success messages are not saved.
            </p>
          ) : (
            <NotificationDrawerList aria-label="Notification list">
              {globalItems.map((item) => (
                <NotificationDrawerListItem key={`global-${item.id}`} variant={mapVariant(item.severity)}>
                  <NotificationDrawerListItemHeader title={item.title} variant={mapVariant(item.severity)}>
                    <Button
                      variant={ButtonVariant.plain}
                      aria-label="Dismiss notification for all users"
                      icon={<TimesIcon />}
                      onClick={() => void dismissGlobalNotification(item.id)}
                    />
                  </NotificationDrawerListItemHeader>
                  <NotificationDrawerListItemBody
                    timestamp={new Date(item.createdAt).toLocaleString()}
                  >
                    {item.message ?? ''}
                    <div className="pf-v6-u-mt-sm">
                      <Button
                        variant={ButtonVariant.link}
                        isInline
                        onClick={() => void dismissGlobalNotification(item.id)}
                      >
                        Acknowledge
                      </Button>
                    </div>
                  </NotificationDrawerListItemBody>
                </NotificationDrawerListItem>
              ))}
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
