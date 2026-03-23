import { useAuthStore } from './store/authStore';

/** Current user has this permission key (also true for admin). Works outside React via the auth store. */
export function hasPermission(permissionKey: string): boolean {
  return useAuthStore.getState().hasPermission(permissionKey);
}

/** Current user has a role with this name. */
export function hasRole(roleName: string): boolean {
  return useAuthStore.getState().hasRole(roleName);
}
