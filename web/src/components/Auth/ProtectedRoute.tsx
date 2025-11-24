import React from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { useAuthStore } from '../../store/authStore';

interface ProtectedRouteProps {
  children: React.ReactNode;
  requiredPermissions?: {
    action: string;
    subject: string;
  }[];
}

const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ 
  children,
  requiredPermissions = []
}) => {
  const { user, loading, hasPermission } = useAuthStore();
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
  
  // Check permissions if specified
  if (requiredPermissions.length > 0) {
    const hasAllPermissions = requiredPermissions.every(
      ({ action, subject }) => hasPermission(action, subject)
    );
    
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