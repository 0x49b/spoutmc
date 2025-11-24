import React from 'react';
import Skeleton from '../UI/Skeleton';

const ServerCardSkeleton: React.FC = () => {
  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm overflow-hidden p-5">
      <div className="flex justify-between items-start mb-4">
        <Skeleton className="h-6 w-32" />
        <Skeleton className="h-6 w-16" variant="text" />
      </div>

      <div className="space-y-3 mb-4">
        <div className="flex items-center justify-between">
          <Skeleton className="h-4 w-16" variant="text" />
          <Skeleton className="h-4 w-20" variant="text" />
        </div>
        <div className="flex items-center justify-between">
          <Skeleton className="h-4 w-16" variant="text" />
          <Skeleton className="h-4 w-24" variant="text" />
        </div>
        <div className="flex items-center justify-between">
          <Skeleton className="h-4 w-16" variant="text" />
          <Skeleton className="h-4 w-20" variant="text" />
        </div>
        <div className="flex items-center justify-between">
          <Skeleton className="h-4 w-16" variant="text" />
          <Skeleton className="h-4 w-32" variant="text" />
        </div>
      </div>

      <div className="pt-3 border-t border-gray-200 dark:border-gray-700">
        <div className="space-y-2">
          <div>
            <div className="flex items-center justify-between mb-1">
              <Skeleton className="h-3 w-20" variant="text" />
              <Skeleton className="h-3 w-10" variant="text" />
            </div>
            <Skeleton className="h-1.5 w-full" />
          </div>

          <div>
            <div className="flex items-center justify-between mb-1">
              <Skeleton className="h-3 w-24" variant="text" />
              <Skeleton className="h-3 w-10" variant="text" />
            </div>
            <Skeleton className="h-1.5 w-full" />
          </div>
        </div>
      </div>
    </div>
  );
};

export default ServerCardSkeleton;
