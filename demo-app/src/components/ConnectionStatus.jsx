import React from 'react';
import { Wifi, WifiOff, RefreshCw, AlertCircle } from 'lucide-react';

const ConnectionStatus = ({ connected, lastUpdate, onRefresh, loading, error }) => {
  return (
    <div className="flex items-center gap-4 text-sm">
      <div className="flex items-center gap-2">
        {error ? (
          <>
            <AlertCircle className="w-4 h-4 text-orange-500" />
            <span className="text-orange-600 font-medium">Error</span>
          </>
        ) : connected ? (
          <>
            <Wifi className="w-4 h-4 text-green-500" />
            <span className="text-green-600 font-medium">Live Updates</span>
          </>
        ) : (
          <>
            <WifiOff className="w-4 h-4 text-red-500" />
            <span className="text-red-600 font-medium">Disconnected</span>
          </>
        )}
      </div>

      {lastUpdate && (
        <div className="text-gray-500">
          Last update: {lastUpdate.toLocaleTimeString()}
        </div>
      )}

      {error && (
        <div className="text-orange-600 text-xs max-w-xs truncate" title={error}>
          {error}
        </div>
      )}
    </div>
  );
};

export default ConnectionStatus;
