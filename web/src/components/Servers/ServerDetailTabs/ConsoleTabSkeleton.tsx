import React from 'react';
import Skeleton from '../../UI/Skeleton';

const ConsoleTabSkeleton: React.FC = () => {
  return (
    <div className="p-6">
      {/* Console Header Skeleton */}
      <div className="flex items-center justify-between mb-4">
        <Skeleton className="h-6 w-32" />
        <div className="flex items-center space-x-2">
          <Skeleton className="h-9 w-24" />
          <Skeleton className="h-9 w-24" />
        </div>
      </div>

      {/* Console Output Skeleton */}
      <div className="bg-gray-900 dark:bg-black rounded-lg p-4 font-mono text-sm space-y-2">
        {[...Array(15)].map((_, index) => (
          <Skeleton
            key={index}
            className="h-4"
            width={`${Math.random() * 40 + 60}%`}
          />
        ))}
      </div>

      {/* Command Input Skeleton */}
      <div className="mt-4">
        <Skeleton className="h-10 w-full" />
      </div>
    </div>
  );
};

export default ConsoleTabSkeleton;
