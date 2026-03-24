import React from 'react';
import {Navigate, useLocation} from 'react-router-dom';
import {useAuthStore} from '../../store/authStore';

interface ProtectedRouteProps {
  children: React.ReactNode;
  /** Permission keys (e.g. auth.user.manage) — user must have all of them (or admin). */
  requiredPermissionKeys?: string[];
  /** Only users with the admin role (not just inherited permissions). */
  requireAdmin?: boolean;
}

const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ 
  children,
  requiredPermissionKeys = [],
  requireAdmin = false
}) => {
  const { user, loading, hasPermission, hasRole } = useAuthStore();
  const location = useLocation();
  
  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary-600"></div>
      </div>
    );
  }
  
  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  if (requireAdmin && !hasRole('admin')) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <h2 className="text-2xl font-bold text-gray-900 mb-2">Access Denied</h2>
          <p className="text-gray-600">This page is only available to administrators.</p>
        </div>
      </div>
    );
  }
  
  if (requiredPermissionKeys.length > 0) {
    const hasAllPermissions = requiredPermissionKeys.every((key) => hasPermission(key));
    
    if (!hasAllPermissions) {
      return (
        <div className="min-h-screen flex items-center justify-center">
          <div className="text-center">
            <h2 className="text-2xl font-bold text-gray-900 mb-2">Access Denied</h2>
            <p className="text-gray-600">
              You don't have permission to access this page.
            </p>
          </div>
        </div>
      );
    }
  }
  
  return <>{children}</>;
};

export default ProtectedRoute;