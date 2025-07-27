import { useState, useEffect, useCallback } from 'react';

const useRemoteConfig = (org, app, env) => {
  const [config, setConfig] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [connected, setConnected] = useState(false);
  const [lastUpdate, setLastUpdate] = useState(null);

  // Fetch initial configuration
  const fetchConfig = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await fetch(`/config/${org}/${app}/${env}`);
      if (!response.ok) {
        throw new Error(`Failed to fetch config: ${response.status}`);
      }
      
      const data = await response.json();
      setConfig(data.config);
      setLastUpdate(new Date());
    } catch (err) {
      setError(err.message);
      console.error('Failed to fetch config:', err);
    } finally {
      setLoading(false);
    }
  }, [org, app, env]);

  // Set up Server-Sent Events for real-time updates
  useEffect(() => {
    if (!org || !app || !env) return;

    // Fetch initial config
    fetchConfig();

    // Set up SSE connection
    const sseUrl = `/events/${org}/${app}/${env}`;
    const eventSource = new EventSource(sseUrl);

    eventSource.onopen = () => {
      setConnected(true);
      setError(null);
    };

    // Handle different event types
    const handleSSEEvent = (event) => {
      try {
        const data = JSON.parse(event.data);

        if (data.config) {
          setConfig(data.config);
          setLastUpdate(new Date());
        }
      } catch (err) {
        console.error('Failed to parse SSE event:', err);
      }
    };

    eventSource.onmessage = handleSSEEvent;

    // Listen for specific event types
    eventSource.addEventListener('connected', () => {
      setConnected(true);
      setError(null);
    });

    eventSource.addEventListener('initial_config', handleSSEEvent);
    eventSource.addEventListener('config_update', handleSSEEvent);

    eventSource.onerror = () => {
      setConnected(false);

      if (eventSource.readyState === EventSource.CLOSED) {
        setError('Connection closed by server');
      } else {
        setError('Connection lost. Attempting to reconnect...');
      }
    };

    // Cleanup on unmount
    return () => {
      eventSource.close();
      setConnected(false);
    };
  }, [org, app, env, fetchConfig]);

  return {
    config,
    loading,
    error,
    connected,
    lastUpdate,
    refetch: fetchConfig
  };
};

export default useRemoteConfig;
