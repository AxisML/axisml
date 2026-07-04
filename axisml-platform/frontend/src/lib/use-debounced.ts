import { useEffect, useState } from "react";

// Debounce a rapidly-changing value (e.g. a search box) so it only propagates to
// a server query after the user pauses, avoiding a request per keystroke.
export function useDebouncedValue<T>(value: T, delayMs = 300): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const timer = window.setTimeout(() => setDebounced(value), delayMs);
    return () => clearTimeout(timer);
  }, [value, delayMs]);
  return debounced;
}
