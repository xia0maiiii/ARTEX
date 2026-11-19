"use client";

import * as React from "react";

import { Card, CardContent } from "@/components/ui/card";
import { ExplorationGraph } from "@/components/exploration-graph";
import { usePoll } from "@/hooks/use-poll";
import { api } from "@/lib/api";
import type { Edge, TaskNode } from "@/lib/types";

export function GraphTab({ taskId }: { taskId: string }) {
  const [nodes, setNodes] = React.useState<TaskNode[]>([]);
  const [edges, setEdges] = React.useState<Edge[]>([]);
  // 上一次图数据的签名:轮询拿到相同数据时跳过 setState,避免整图无谓重建(拖动时
  // 才不会被 20s 轮询打断而顿挫)。只取影响渲染的字段。
  const sigRef = React.useRef("");

  // 详情页以 key={taskId} 挂载本标签,taskId 在实例内固定,换任务即重新挂载。
  usePoll(
    () =>
      api
        .explorationGraph(taskId)
        .then((g) => {
          const ns = g.nodes ?? [];
          const es = g.edges ?? [];
          const sig = JSON.stringify([
            ns.map((n) => [n.id, n.type, n.state, n.priority, n.payload]),
            es.map((e) => [e.src, e.dst, e.rel]),
          ]);
          if (sig === sigRef.current) return; // 无变化 → 不重建
          sigRef.current = sig;
          setNodes(ns);
          setEdges(es);
        })
        .catch(() => {
          /* keep last good data */
        }),
    20000,
  );

  return (
    <Card>
      <CardContent className="p-0">
        <ExplorationGraph nodes={nodes} edges={edges} className="h-[72vh]" />
      </CardContent>
    </Card>
  );
}
