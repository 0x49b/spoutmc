import React from 'react';
import Skeleton from '../../UI/Skeleton';

const OverviewTabSkeleton: React.FC = () => {
  return (
    <div className="p-6">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
        {/* Server Information Skeleton */}
        <div>
          <Skeleton className="h-6 w-40 mb-4" />
          <div className="space-y-3">
            <div className="flex justify-between">
              <Skeleton className="h-4 w-20" variant="text" />
              <Skeleton className="h-4 w-24" variant="text" />
            </div>
            <div className="flex justify-between">
              <Skeleton className="h-4 w-20" variant="text" />
              <Skeleton className="h-4 w-32" variant="text" />
            </div>
            <div className="flex justify-between">
              <Skeleton className="h-4 w-20" variant="text" />
              <Skeleton className="h-4 w-28" variant="text" />
            </div>
          </div>
        </div>

        {/* Resource Usage Skeleton */}
        <div>
          <Skeleton className="h-6 w-32 mb-4" />
          <div className="space-y-5">
            <div>
              <div className="flex items-center justify-between mb-2">
                <Skeleton className="h-4 w-24" variant="text" />
                <Skeleton className="h-4 w-12" variant="text" />
              </div>
              <Skeleton className="h-2 w-full" />
            </div>

            <div>
              <div className="flex items-center justify-between mb-2">
                <Skeleton className="h-4 w-28" variant="text" />
                <Skeleton className="h-4 w-24" variant="text" />
              </div>
              <Skeleton className="h-2 w-full" />
            </div>
          </div>
        </div>
      </div>

      {/* Environment Variables Skeleton */}
      <div className="mb-6">
        <Skeleton className="h-6 w-48 mb-4" />
        <div className="space-y-2 p-4 bg-gray-800 rounded border border-gray-700">
          <div className="flex justify-between">
            <Skeleton className="h-4 w-32" variant="text" />
            <Skeleton className="h-4 w-40" variant="text" />
          </div>
          <div className="flex justify-between">
            <Skeleton className="h-4 w-36" variant="text" />
            <Skeleton className="h-4 w-32" variant="text" />
          </div>
          <div className="flex justify-between">
            <Skeleton className="h-4 w-28" variant="text" />
            <Skeleton className="h-4 w-36" variant="text" />
          </div>
          <div className="flex justify-between">
            <Skeleton className="h-4 w-40" variant="text" />
            <Skeleton className="h-4 w-28" variant="text" />
          </div>
          <div className="flex justify-between">
            <Skeleton className="h-4 w-32" variant="text" />
            <Skeleton className="h-4 w-44" variant="text" />
          </div>
        </div>
      </div>
    </div>
  );
};

export default OverviewTabSkeleton;
