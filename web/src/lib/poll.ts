// Pure polling scheduler behind `usePoll` (src/hooks/use-poll.ts). It has no React or
// DOM dependency so `poll.test.mjs` can drive it with a fake clock.
//
// Rules:
//   • The next tick is scheduled only after the previous run settles, so runs never
//     overlap no matter how slow the backend is.
//   • While hidden nothing is scheduled; becoming visible runs once immediately.
//   • refresh() runs now, or — if a run is in flight — exactly once right after it.

export type PollEnv = {
  isHidden: () => boolean;
  setTimer: (callback: () => void, ms: number) => unknown;
  clearTimer: (handle: unknown) => void;
};

export type PollLoop = {
  /** (Re)start ticking. A run already in flight is reused instead of starting another. */
  start: (intervalMs: number, immediate: boolean) => void;
  /** Stop scheduling. A run in flight finishes but schedules nothing. */
  stop: () => void;
  /** Run now (even when stopped or hidden); resolves once that run has settled. */
  refresh: () => Promise<void>;
  /** Call on `visibilitychange`. */
  visibilityChanged: () => void;
};

const noop = () => undefined;

export function createPollLoop(run: () => unknown, env: PollEnv): PollLoop {
  let stopped = true;
  let intervalMs = 0;
  let timer: unknown = null;
  let inFlight: Promise<void> | null = null;
  let queued: Promise<void> | null = null;

  const clearTimer = () => {
    if (timer === null) return;
    env.clearTimer(timer);
    timer = null;
  };

  const schedule = () => {
    clearTimer();
    // Hidden: stay paused until visibilityChanged() resumes the loop.
    if (stopped || env.isHidden()) return;
    timer = env.setTimer(tick, intervalMs);
  };

  const execute = (): Promise<void> => {
    clearTimer();
    let result: unknown;
    try {
      result = run();
    } catch {
      result = undefined; // callers own their error handling; keep the loop alive
    }
    const settled = Promise.resolve(result)
      .then(noop, noop)
      .then(() => {
        inFlight = null;
        schedule();
      });
    inFlight = settled;
    return settled;
  };

  function tick() {
    timer = null;
    if (stopped || env.isHidden() || inFlight !== null) return;
    void execute();
  }

  return {
    start(nextIntervalMs, immediate) {
      stopped = false;
      intervalMs = nextIntervalMs;
      // A run still in flight (e.g. across a restart) schedules the next tick itself.
      if (inFlight !== null) return;
      if (immediate && !env.isHidden()) void execute();
      else schedule();
    },
    stop() {
      stopped = true;
      clearTimer();
    },
    refresh() {
      if (inFlight === null) return execute();
      queued ??= inFlight.then(() => {
        queued = null;
        return inFlight ?? execute();
      });
      return queued;
    },
    visibilityChanged() {
      if (stopped) return;
      if (env.isHidden()) clearTimer();
      else if (inFlight === null) void execute();
    },
  };
}

/**
 * Returns `prev` when `next` serializes identically, so `setState((prev) => keepIfUnchanged(prev, next))`
 * bails out of re-rendering when a poll brings back the same data. Comparing with the
 * current state (rather than a cached signature) stays correct after optimistic local edits.
 */
export function keepIfUnchanged<T>(prev: T, next: T): T {
  if (Object.is(prev, next)) return prev;
  return JSON.stringify(prev) === JSON.stringify(next) ? prev : next;
}

/**
 * List variant of keepIfUnchanged: rows whose data did not change keep their previous
 * object (matched by `key`), and the previous array itself is returned when every row
 * and the order are the same. Lets a selected row keep its identity while others move.
 */
export function keepUnchangedItems<T>(prev: T[], next: T[], key: (item: T) => unknown): T[] {
  const byKey = new Map(prev.map((item) => [key(item), item]));
  let same = prev.length === next.length;
  const merged = next.map((item, index) => {
    const old = byKey.get(key(item));
    const kept = old !== undefined && keepIfUnchanged(old, item) === old ? old : item;
    if (kept !== prev[index]) same = false;
    return kept;
  });
  return same ? prev : merged;
}
