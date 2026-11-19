import { createPollLoop, keepIfUnchanged, keepUnchangedItems } from "./poll.ts";
import assert from "node:assert/strict";
import test from "node:test";

// Manual clock: timers fire only when advance() passes their due time.
function fakeEnv() {
  let now = 0;
  let nextId = 1;
  const timers = new Map();
  const env = {
    hidden: false,
    isHidden: () => env.hidden,
    setTimer: (callback, ms) => {
      const id = nextId++;
      timers.set(id, { callback, at: now + ms });
      return id;
    },
    clearTimer: (id) => timers.delete(id),
    pending: () => timers.size,
    advance: async (ms) => {
      const end = now + ms;
      for (;;) {
        const due = [...timers.entries()].filter(([, t]) => t.at <= end).sort(([, a], [, b]) => a.at - b.at)[0];
        if (!due) break;
        const [id, t] = due;
        timers.delete(id);
        now = t.at;
        t.callback();
        await flush();
      }
      now = end;
    },
  };
  return env;
}

const flush = () => new Promise((resolve) => setImmediate(resolve));

// run() whose calls stay pending until resolved by the test.
function controlledRun() {
  const calls = [];
  const run = () =>
    new Promise((resolve, reject) => {
      calls.push({ resolve, reject });
    });
  return { run, calls };
}

test("immediate start runs once and waits for the call to settle before scheduling", async () => {
  const env = fakeEnv();
  const { run, calls } = controlledRun();
  const loop = createPollLoop(run, env);
  loop.start(1000, true);
  assert.equal(calls.length, 1);
  await env.advance(10_000); // a slow request must not cause overlapping calls
  assert.equal(calls.length, 1);
  assert.equal(env.pending(), 0);
  calls[0].resolve();
  await flush();
  assert.equal(env.pending(), 1);
  await env.advance(999);
  assert.equal(calls.length, 1);
  await env.advance(1);
  assert.equal(calls.length, 2);
});

test("non-immediate start waits one interval", async () => {
  const env = fakeEnv();
  let count = 0;
  const loop = createPollLoop(() => count++, env);
  loop.start(500, false);
  assert.equal(count, 0);
  await env.advance(500);
  assert.equal(count, 1);
  await env.advance(500);
  assert.equal(count, 2);
});

test("hidden pauses polling and becoming visible runs once immediately", async () => {
  const env = fakeEnv();
  let count = 0;
  const loop = createPollLoop(() => count++, env);
  loop.start(1000, true);
  await flush();
  assert.equal(count, 1);
  env.hidden = true;
  loop.visibilityChanged();
  assert.equal(env.pending(), 0);
  await env.advance(60_000);
  assert.equal(count, 1);
  env.hidden = false;
  loop.visibilityChanged();
  assert.equal(count, 2);
  await flush();
  await env.advance(1000);
  assert.equal(count, 3);
});

test("a tick that fires while hidden is skipped until visible", async () => {
  const env = fakeEnv();
  let count = 0;
  const loop = createPollLoop(() => count++, env);
  loop.start(1000, false);
  env.hidden = true; // no visibilitychange delivered yet
  await env.advance(5000);
  assert.equal(count, 0);
  assert.equal(env.pending(), 0);
  env.hidden = false;
  loop.visibilityChanged();
  assert.equal(count, 1);
});

test("starting while hidden defers the first run until visible", async () => {
  const env = fakeEnv();
  env.hidden = true;
  let count = 0;
  const loop = createPollLoop(() => count++, env);
  loop.start(1000, true);
  await env.advance(5000);
  assert.equal(count, 0);
  env.hidden = false;
  loop.visibilityChanged();
  assert.equal(count, 1);
});

test("stop cancels the timer and an in-flight call schedules nothing afterwards", async () => {
  const env = fakeEnv();
  const { run, calls } = controlledRun();
  const loop = createPollLoop(run, env);
  loop.start(1000, true);
  loop.stop();
  calls[0].resolve();
  await flush();
  assert.equal(env.pending(), 0);
  await env.advance(10_000);
  assert.equal(calls.length, 1);
  loop.visibilityChanged(); // ignored while stopped
  assert.equal(calls.length, 1);
});

test("restarting while a call is in flight does not start a second one", async () => {
  const env = fakeEnv();
  const { run, calls } = controlledRun();
  const loop = createPollLoop(run, env);
  loop.start(1000, true);
  loop.stop();
  loop.start(2000, true);
  assert.equal(calls.length, 1);
  calls[0].resolve();
  await flush();
  await env.advance(1999);
  assert.equal(calls.length, 1);
  await env.advance(1);
  assert.equal(calls.length, 2); // the new interval applies
});

test("refresh runs now and reschedules from that point", async () => {
  const env = fakeEnv();
  let count = 0;
  const loop = createPollLoop(() => count++, env);
  loop.start(1000, true);
  await flush();
  await env.advance(600);
  await loop.refresh();
  assert.equal(count, 2);
  await env.advance(999);
  assert.equal(count, 2);
  await env.advance(1);
  assert.equal(count, 3);
});

test("refresh during an in-flight call queues exactly one follow-up run", async () => {
  const env = fakeEnv();
  const { run, calls } = controlledRun();
  const loop = createPollLoop(run, env);
  loop.start(1000, true);
  let refreshed = 0;
  const first = loop.refresh().then(() => refreshed++);
  const second = loop.refresh().then(() => refreshed++);
  assert.equal(calls.length, 1);
  calls[0].resolve();
  await flush();
  assert.equal(calls.length, 2); // one follow-up, not two, and not overlapping
  assert.equal(refreshed, 0);
  calls[1].resolve();
  await Promise.all([first, second]);
  assert.equal(refreshed, 2);
  assert.equal(calls.length, 2);
});

test("refresh works while stopped and does not resume polling", async () => {
  const env = fakeEnv();
  let count = 0;
  const loop = createPollLoop(() => count++, env);
  await loop.refresh();
  assert.equal(count, 1);
  assert.equal(env.pending(), 0);
});

test("errors and rejections keep the loop alive", async () => {
  const env = fakeEnv();
  let count = 0;
  const loop = createPollLoop(() => {
    count++;
    if (count === 1) throw new Error("sync");
    return Promise.reject(new Error("async"));
  }, env);
  loop.start(100, true);
  await flush();
  await env.advance(100);
  await env.advance(100);
  assert.equal(count, 3);
});

test("keepIfUnchanged keeps the previous reference for equal data", () => {
  const prev = [{ id: 1, status: "open" }];
  assert.equal(keepIfUnchanged(prev, [{ id: 1, status: "open" }]), prev);
  const changed = [{ id: 1, status: "done" }];
  assert.equal(keepIfUnchanged(prev, changed), changed);
  assert.equal(keepIfUnchanged(null, null), null);
  const next = { a: 1 };
  assert.equal(keepIfUnchanged(null, next), next);
});

test("keepUnchangedItems reuses unchanged rows and the array itself when nothing moved", () => {
  const a = { id: 1, title: "a" };
  const b = { id: 2, title: "b" };
  const prev = [a, b];
  const byId = (row) => row.id;
  assert.equal(keepUnchangedItems(prev, [{ ...a }, { ...b }], byId), prev);

  const changed = keepUnchangedItems(prev, [{ id: 1, title: "a2" }, { ...b }], byId);
  assert.notEqual(changed, prev);
  assert.equal(changed[1], b); // the untouched row keeps its identity
  assert.deepEqual(changed[0], { id: 1, title: "a2" });

  const reordered = keepUnchangedItems(prev, [{ ...b }, { ...a }], byId);
  assert.notEqual(reordered, prev);
  assert.equal(reordered[0], b);
  assert.equal(reordered[1], a);

  const shorter = keepUnchangedItems(prev, [{ ...a }], byId);
  assert.notEqual(shorter, prev);
  assert.equal(shorter[0], a);
});
