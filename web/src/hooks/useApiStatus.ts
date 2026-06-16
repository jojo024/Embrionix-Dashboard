import { useEffect, useState } from 'react';
import { api } from '../api/client';

export function useApiStatus() {
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    let interval: number;

    const checkApi = async () => {
      try {
        await api.health();
        setConnected(true);
      } catch {
        setConnected(false);
      }
    };

    checkApi();
    interval = window.setInterval(checkApi, 10000);

    return () => clearInterval(interval);
  }, []);

  return connected;
}
