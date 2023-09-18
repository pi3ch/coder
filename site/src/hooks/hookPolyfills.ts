/**
 * @file For defining DIY versions of official React hooks that have not been
 * released yet.
 *
 * These hooks should be deleted as soon as the official versions are available.
 * They also do have the same ESLinter exceptions baked in that the official
 * hooks do.
 */
import { useCallback, useEffect, useRef } from "react";

/**
 * A DIY version of useEffectEvent.
 *
 * Works like useCallback, except that it doesn't take a dependency array, and
 * always returns out a stable function on every single render. The returned-out
 * function is always able to "see" the most up-to-date version of the callback
 * passed in.
 *
 * Should only be used as a last resort where useCallback does not work, and you
 * still need to avoid dependency array violations.
 *
 * @see {@link https://react.dev/reference/react/experimental_useEffectEvent}
 */
export function useEffectEvent<TArgs extends unknown[], TReturn = unknown>(
  callback: (...args: TArgs) => TReturn,
) {
  const callbackRef = useRef(callback);
  useEffect(() => {
    callbackRef.current = callback;
  }, [callback]);

  return useCallback((...args: TArgs): TReturn => {
    return callbackRef.current(...args);
  }, []);
}
