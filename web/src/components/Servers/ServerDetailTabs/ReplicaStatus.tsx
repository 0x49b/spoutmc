import React from 'react';
import { motion } from 'framer-motion';
import { Server } from '../../../types';

interface ReplicaStatusProps {
  server: Server;
}

export const ReplicaStatus: React.FC<ReplicaStatusProps> = ({ server }) => {
  const segments = React.useMemo(() => {
    const total = server.desiredReplicas;
    const ready = server.readyReplicas;
    const updating = server.updatingReplicas;
    const available = server.availableReplicas;

    const segmentSize = 360 / total;
    
    return Array.from({ length: total }).map((_, index) => {
      let status: 'ready' | 'updating' | 'pending' = 'ready';
      
      if (index < ready) {
        status = 'ready';
      } else if (index < (ready + updating)) {
        status = 'updating';
      }

      return {
        rotation: index * segmentSize,
        status
      };
    });
  }, [server.desiredReplicas, server.readyReplicas, server.updatingReplicas, server.availableReplicas]);

  return (
    <div className="relative w-32 h-32">
      <div className="absolute inset-0 flex items-center justify-center">
        <div className="text-center">
          <div className="text-2xl font-bold text-gray-900 dark:text-white">
            {server.readyReplicas}
          </div>
          <div className="text-sm text-gray-500 dark:text-gray-400">
            Pods
          </div>
        </div>
      </div>
      <svg
        viewBox="0 0 100 100"
        className="transform -rotate-90 w-full h-full"
      >
        {segments.map((segment, index) => (
          <motion.path
            key={index}
            initial={{ pathLength: 0 }}
            animate={{ pathLength: 1 }}
            transition={{ duration: 0.5, delay: index * 0.1 }}
            d={describeArc(50, 50, 40, segment.rotation, segment.rotation + (360 / segments.length) - 2)}
            fill="none"
            strokeWidth="8"
            strokeLinecap="round"
            className={
              segment.status === 'ready'
                ? 'stroke-primary-500 dark:stroke-primary-400'
                : segment.status === 'updating'
                ? 'stroke-primary-300 dark:stroke-primary-600'
                : 'stroke-gray-200 dark:stroke-gray-700'
            }
          />
        ))}
      </svg>
    </div>
  );
};

// Helper functions to draw the arc segments
function polarToCartesian(centerX: number, centerY: number, radius: number, angleInDegrees: number) {
  const angleInRadians = (angleInDegrees - 90) * Math.PI / 180.0;
  return {
    x: centerX + (radius * Math.cos(angleInRadians)),
    y: centerY + (radius * Math.sin(angleInRadians))
  };
}

function describeArc(x: number, y: number, radius: number, startAngle: number, endAngle: number) {
  const start = polarToCartesian(x, y, radius, endAngle);
  const end = polarToCartesian(x, y, radius, startAngle);
  const largeArcFlag = endAngle - startAngle <= 180 ? "0" : "1";
  return [
    "M", start.x, start.y,
    "A", radius, radius, 0, largeArcFlag, 0, end.x, end.y
  ].join(" ");
}

export default ReplicaStatus;