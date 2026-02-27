import { useCallback, useRef, useState, type ReactNode } from "react";

type SplitDirection = "horizontal" | "vertical";

interface SplitViewProps {
  direction?: SplitDirection;
  initialRatio?: number;
  minRatio?: number;
  maxRatio?: number;
  primary: ReactNode;
  secondary: ReactNode;
}

export function SplitView({
  direction = "horizontal",
  initialRatio = 0.65,
  minRatio = 0.2,
  maxRatio = 0.8,
  primary,
  secondary,
}: SplitViewProps) {
  const [ratio, setRatio] = useState(initialRatio);
  const containerRef = useRef<HTMLDivElement>(null);
  const dragging = useRef(false);

  const onMouseDown = useCallback(() => {
    dragging.current = true;
    document.body.style.cursor =
      direction === "horizontal" ? "col-resize" : "row-resize";
    document.body.style.userSelect = "none";
  }, [direction]);

  const onMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!dragging.current || !containerRef.current) return;

      const rect = containerRef.current.getBoundingClientRect();
      let newRatio: number;

      if (direction === "horizontal") {
        newRatio = (e.clientX - rect.left) / rect.width;
      } else {
        newRatio = (e.clientY - rect.top) / rect.height;
      }

      setRatio(Math.max(minRatio, Math.min(maxRatio, newRatio)));
    },
    [direction, minRatio, maxRatio]
  );

  const onMouseUp = useCallback(() => {
    dragging.current = false;
    document.body.style.cursor = "";
    document.body.style.userSelect = "";
  }, []);

  const attachListeners = useCallback(() => {
    document.addEventListener("mousemove", onMouseMove);
    document.addEventListener("mouseup", onMouseUp);
    return () => {
      document.removeEventListener("mousemove", onMouseMove);
      document.removeEventListener("mouseup", onMouseUp);
    };
  }, [onMouseMove, onMouseUp]);

  const handleMouseDown = useCallback(() => {
    onMouseDown();
    const cleanup = attachListeners();
    const origUp = onMouseUp;
    const wrappedUp = () => {
      origUp();
      cleanup();
    };
    document.removeEventListener("mouseup", onMouseUp);
    document.addEventListener("mouseup", wrappedUp, { once: true });
  }, [onMouseDown, onMouseUp, attachListeners]);

  const isHorizontal = direction === "horizontal";
  const primarySize = `${ratio * 100}%`;
  const secondarySize = `${(1 - ratio) * 100}%`;

  return (
    <div
      ref={containerRef}
      className={`flex ${isHorizontal ? "flex-row" : "flex-col"} h-full w-full`}
    >
      <div
        className="overflow-hidden"
        style={isHorizontal ? { width: primarySize } : { height: primarySize }}
      >
        {primary}
      </div>

      <div
        onMouseDown={handleMouseDown}
        className={`shrink-0 ${
          isHorizontal
            ? "w-1 cursor-col-resize hover:bg-blue-500/40"
            : "h-1 cursor-row-resize hover:bg-blue-500/40"
        } bg-gray-700 transition-colors`}
      />

      <div
        className="overflow-hidden"
        style={
          isHorizontal ? { width: secondarySize } : { height: secondarySize }
        }
      >
        {secondary}
      </div>
    </div>
  );
}
