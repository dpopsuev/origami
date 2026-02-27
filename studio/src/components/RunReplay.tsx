import { useCallback, useEffect, useRef, useState } from "react";

interface ReplayEvent {
  type: string;
  node?: string;
  edge?: string;
  walker?: string;
  ts: string;
  elapsed_ms?: number;
}

interface RunReplayProps {
  events: ReplayEvent[];
  speed: number;
  onEvent: (event: ReplayEvent) => void;
}

export function useRunReplay({ events, speed, onEvent }: RunReplayProps) {
  const [playing, setPlaying] = useState(false);
  const [currentIndex, setCurrentIndex] = useState(0);
  const [progress, setProgress] = useState(0);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const play = useCallback(() => setPlaying(true), []);
  const pause = useCallback(() => setPlaying(false), []);

  const reset = useCallback(() => {
    setPlaying(false);
    setCurrentIndex(0);
    setProgress(0);
  }, []);

  const seekTo = useCallback(
    (index: number) => {
      const clamped = Math.max(0, Math.min(events.length - 1, index));
      setCurrentIndex(clamped);
      setProgress(events.length > 0 ? clamped / events.length : 0);
    },
    [events.length]
  );

  useEffect(() => {
    if (!playing || currentIndex >= events.length) {
      if (currentIndex >= events.length) setPlaying(false);
      return;
    }

    const event = events[currentIndex];
    onEvent(event);

    let delayMs = 200;
    if (currentIndex < events.length - 1) {
      const currentTs = new Date(event.ts).getTime();
      const nextTs = new Date(events[currentIndex + 1].ts).getTime();
      delayMs = Math.max(50, (nextTs - currentTs) / speed);
    }

    timerRef.current = setTimeout(() => {
      setCurrentIndex((i) => i + 1);
      setProgress((currentIndex + 1) / events.length);
    }, delayMs);

    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, [playing, currentIndex, events, speed, onEvent]);

  return {
    playing,
    currentIndex,
    progress,
    totalEvents: events.length,
    play,
    pause,
    reset,
    seekTo,
  };
}

interface ReplayControlsProps {
  playing: boolean;
  progress: number;
  currentIndex: number;
  totalEvents: number;
  speed: number;
  onPlay: () => void;
  onPause: () => void;
  onReset: () => void;
  onSpeedChange: (speed: number) => void;
  onSeek: (index: number) => void;
}

export function ReplayControls({
  playing,
  progress,
  currentIndex,
  totalEvents,
  speed,
  onPlay,
  onPause,
  onReset,
  onSpeedChange,
  onSeek,
}: ReplayControlsProps) {
  return (
    <div className="flex items-center gap-3 px-4 py-2 bg-gray-800 border-t border-gray-700">
      <button
        onClick={playing ? onPause : onPlay}
        className="text-sm px-3 py-1 rounded bg-gray-700 hover:bg-gray-600"
      >
        {playing ? "⏸" : "▶"}
      </button>
      <button
        onClick={onReset}
        className="text-sm px-3 py-1 rounded bg-gray-700 hover:bg-gray-600"
      >
        ⏹
      </button>

      <div className="flex-1 relative h-1 bg-gray-700 rounded cursor-pointer"
        onClick={(e) => {
          const rect = e.currentTarget.getBoundingClientRect();
          const pct = (e.clientX - rect.left) / rect.width;
          onSeek(Math.floor(pct * totalEvents));
        }}
      >
        <div
          className="absolute h-full bg-blue-500 rounded"
          style={{ width: `${progress * 100}%` }}
        />
      </div>

      <span className="text-xs text-gray-500 font-mono w-20 text-right">
        {currentIndex}/{totalEvents}
      </span>

      <select
        value={speed}
        onChange={(e) => onSpeedChange(parseFloat(e.target.value))}
        className="bg-gray-700 text-xs rounded px-2 py-1 border-none focus:outline-none"
      >
        <option value={0.5}>0.5x</option>
        <option value={1}>1x</option>
        <option value={2}>2x</option>
        <option value={4}>4x</option>
        <option value={10}>10x</option>
      </select>
    </div>
  );
}
