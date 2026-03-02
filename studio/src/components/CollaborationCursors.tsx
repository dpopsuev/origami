import { useEffect, useState } from "react";

interface CursorPosition {
  userId: string;
  userName: string;
  element: string;
  x: number;
  y: number;
  selectedNode?: string;
  lastActive: string;
}

const ELEMENT_COLORS: Record<string, string> = {
  fire: "#DC143C",
  water: "#007BA7",
  earth: "#0047AB",
  air: "#FFBF00",
  diamond: "#0F52BA",
  lightning: "#DC143C",
};

interface CollaborationCursorsProps {
  currentUserId: string;
  wsUrl?: string;
}

export function CollaborationCursors({
  currentUserId,
  wsUrl = "/ws/presence",
}: CollaborationCursorsProps) {
  const [cursors, setCursors] = useState<Map<string, CursorPosition>>(new Map());

  useEffect(() => {
    let ws: WebSocket | null = null;

    try {
      ws = new WebSocket(wsUrl);

      ws.onmessage = (msg) => {
        try {
          const pos: CursorPosition = JSON.parse(msg.data);
          if (pos.userId === currentUserId) return;

          setCursors((prev) => {
            const next = new Map(prev);
            next.set(pos.userId, pos);
            return next;
          });
        } catch {
          // skip malformed
        }
      };

      ws.onclose = () => {
        setCursors(new Map());
      };
    } catch {
      // WebSocket not available
    }

    return () => ws?.close();
  }, [currentUserId, wsUrl]);

  const staleThreshold = 30_000;
  const now = Date.now();

  return (
    <>
      {Array.from(cursors.values())
        .filter((c) => now - new Date(c.lastActive).getTime() < staleThreshold)
        .map((cursor) => {
          const color = ELEMENT_COLORS[cursor.element] || "#888";
          return (
            <div
              key={cursor.userId}
              className="absolute pointer-events-none z-50 transition-all duration-150"
              style={{
                left: cursor.x,
                top: cursor.y,
                transform: "translate(-2px, -2px)",
              }}
            >
              <svg width="16" height="20" viewBox="0 0 16 20">
                <path
                  d="M0 0L16 12L8 12L4 20L0 0Z"
                  fill={color}
                  opacity="0.8"
                />
              </svg>
              <div
                className="text-[9px] px-1 py-0.5 rounded mt-0.5 whitespace-nowrap"
                style={{ backgroundColor: color, color: "#fff" }}
              >
                {cursor.userName}
              </div>
              {cursor.selectedNode && (
                <div className="text-[8px] text-gray-400 mt-0.5">
                  @ {cursor.selectedNode}
                </div>
              )}
            </div>
          );
        })}
    </>
  );
}
