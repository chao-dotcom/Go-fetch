import { useEffect, useRef } from 'react';

export function usePolling(callback: () => void, interval: number = 5000, enabled: boolean = true) {
  const savedCallback = useRef<() => void>();

  useEffect(() => {
    savedCallback.current = callback;
  }, [callback]);

  useEffect(() => {
    if (!enabled) return;

    const tick = () => {
      savedCallback.current?.();
    };

    tick(); // Call immediately
    const id = setInterval(tick, interval);
    return () => clearInterval(id);
  }, [interval, enabled]);
}

