import * as React from "react";

import { createPollLoop, type PollEnv, type PollLoop } from "@/lib/poll";

export type UsePollOptions = {
  /** Poll only while true (default true). Toggling restarts the loop. */
  enabled?: boolean;
  /** Run as soon as the loop starts instead of after the first interval (default true). */
  immediate?: boolean;
};

const browserEnv: PollEnv = {
  isHidden: () => typeof document !== "undefined" && document.hidden,
  setTimer: (callback, ms) => window.setTimeout(callback, ms),
  clearTimer: (handle) => window.clearTimeout(handle as number),
};

/**
 * Calls `fn` every `intervalMs` without overlapping: the next call is scheduled only
 * after the previous one settles. Pauses while the tab is hidden and runs once as soon
 * as it becomes visible again. `fn` is read through a ref, so a new callback identity
 * does not restart the loop; only `intervalMs` / `enabled` do.
 *
 * Returns `refresh()`, which runs `fn` now (or right after an in-flight call) and
 * resolves when it settles. It works even while polling is disabled.
 */
export function usePoll(
  fn: () => unknown,
  intervalMs: number,
  { enabled = true, immediate = true }: UsePollOptions = {},
): () => Promise<void> {
  const latest = React.useRef({ fn, immediate });
  React.useLayoutEffect(() => {
    latest.current = { fn, immediate };
  });

  const loopRef = React.useRef<PollLoop | null>(null);
  const getLoop = React.useCallback(() => {
    loopRef.current ??= createPollLoop(() => latest.current.fn(), browserEnv);
    return loopRef.current;
  }, []);

  React.useEffect(() => {
    if (!enabled) return;
    const loop = getLoop();
    const onVisibilityChange = () => loop.visibilityChanged();
    document.addEventListener("visibilitychange", onVisibilityChange);
    loop.start(intervalMs, latest.current.immediate);
    return () => {
      document.removeEventListener("visibilitychange", onVisibilityChange);
      loop.stop();
    };
  }, [enabled, intervalMs, getLoop]);

  return React.useCallback(() => getLoop().refresh(), [getLoop]);
}
