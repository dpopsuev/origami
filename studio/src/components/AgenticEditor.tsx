import { useCallback, useRef, useState } from "react";

interface ChatMessage {
  role: "user" | "assistant" | "system";
  content: string;
  timestamp: string;
  operations?: GraphOperation[];
}

interface GraphOperation {
  type: "add_node" | "remove_node" | "add_edge" | "remove_edge" | "set_condition" | "move_to_zone" | "set_walker";
  target: string;
  params?: Record<string, unknown>;
}

interface OptimizationSuggestion {
  id: string;
  description: string;
  rationale: string;
  operations: GraphOperation[];
  impact: "high" | "medium" | "low";
}

interface AgenticEditorProps {
  enabled: boolean;
  onToggle: (enabled: boolean) => void;
  onExecuteOperations: (ops: GraphOperation[]) => void;
  onUndo: () => void;
  suggestions?: OptimizationSuggestion[];
  onAcceptSuggestion?: (id: string) => void;
  onRejectSuggestion?: (id: string) => void;
}

export function AgenticEditor({
  enabled,
  onToggle,
  onExecuteOperations,
  onUndo,
  suggestions = [],
  onAcceptSuggestion,
  onRejectSuggestion,
}: AgenticEditorProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState("");
  const [processing, setProcessing] = useState(false);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, []);

  const handleSubmit = useCallback(async () => {
    if (!input.trim() || processing) return;

    const userMsg: ChatMessage = {
      role: "user",
      content: input.trim(),
      timestamp: new Date().toISOString(),
    };

    setMessages((prev) => [...prev, userMsg]);
    setInput("");
    setProcessing(true);

    try {
      const resp = await fetch("/api/agentic/intent", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ intent: userMsg.content }),
      });

      if (resp.ok) {
        const data = await resp.json();
        const ops: GraphOperation[] = data.operations || [];

        const assistantMsg: ChatMessage = {
          role: "assistant",
          content: data.explanation || `Executing ${ops.length} operation(s)...`,
          timestamp: new Date().toISOString(),
          operations: ops,
        };

        setMessages((prev) => [...prev, assistantMsg]);
        onExecuteOperations(ops);
      } else {
        setMessages((prev) => [
          ...prev,
          {
            role: "system",
            content: "Failed to process intent. API returned an error.",
            timestamp: new Date().toISOString(),
          },
        ]);
      }
    } catch {
      setMessages((prev) => [
        ...prev,
        {
          role: "system",
          content: "Failed to connect to agentic API.",
          timestamp: new Date().toISOString(),
        },
      ]);
    } finally {
      setProcessing(false);
      setTimeout(scrollToBottom, 100);
    }
  }, [input, processing, onExecuteOperations, scrollToBottom]);

  if (!enabled) {
    return (
      <button
        onClick={() => onToggle(true)}
        className="flex items-center gap-2 px-3 py-1.5 text-xs bg-gray-800 hover:bg-gray-700 rounded border border-gray-600"
      >
        <span className="text-purple-400">AI</span> Assistant
      </button>
    );
  }

  return (
    <div className="w-80 bg-gray-900 border-l border-gray-700 flex flex-col h-full">
      <div className="flex items-center justify-between px-3 py-2 border-b border-gray-700">
        <div className="flex items-center gap-2">
          <span className="text-purple-400 text-sm">AI</span>
          <span className="text-sm font-medium">Agentic Editor</span>
        </div>
        <div className="flex items-center gap-1">
          {suggestions.length > 0 && (
            <button
              onClick={() => setShowSuggestions(!showSuggestions)}
              className="text-xs px-2 py-0.5 bg-yellow-900/50 text-yellow-400 rounded"
            >
              {suggestions.length} suggestions
            </button>
          )}
          <button onClick={onUndo} className="text-xs text-gray-500 hover:text-white px-1">
            ↩
          </button>
          <button
            onClick={() => onToggle(false)}
            className="text-gray-500 hover:text-white text-sm px-1"
          >
            ✕
          </button>
        </div>
      </div>

      {showSuggestions && suggestions.length > 0 && (
        <div className="border-b border-gray-700 p-2 space-y-2 max-h-48 overflow-y-auto">
          {suggestions.map((s) => (
            <div key={s.id} className="p-2 bg-gray-800 rounded text-xs">
              <div className="flex items-center gap-1 mb-1">
                <span
                  className={`w-1.5 h-1.5 rounded-full ${
                    s.impact === "high"
                      ? "bg-red-400"
                      : s.impact === "medium"
                      ? "bg-yellow-400"
                      : "bg-green-400"
                  }`}
                />
                <span className="text-gray-300">{s.description}</span>
              </div>
              <div className="text-gray-500 text-[10px] mb-1">{s.rationale}</div>
              <div className="flex gap-1">
                <button
                  onClick={() => onAcceptSuggestion?.(s.id)}
                  className="text-[10px] px-2 py-0.5 bg-green-900 text-green-300 rounded"
                >
                  Accept
                </button>
                <button
                  onClick={() => onRejectSuggestion?.(s.id)}
                  className="text-[10px] px-2 py-0.5 bg-gray-700 text-gray-400 rounded"
                >
                  Reject
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      <div className="flex-1 overflow-y-auto p-3 space-y-2">
        {messages.length === 0 && (
          <div className="text-gray-600 text-xs text-center mt-8">
            Describe what you want to build or modify.
            <br />
            <span className="text-gray-700">
              "Add a retry loop around triage with max 3 attempts"
            </span>
          </div>
        )}
        {messages.map((msg, i) => (
          <div
            key={i}
            className={`text-xs p-2 rounded ${
              msg.role === "user"
                ? "bg-blue-900/30 text-blue-200 ml-4"
                : msg.role === "assistant"
                ? "bg-gray-800 text-gray-300 mr-4"
                : "bg-red-900/20 text-red-300"
            }`}
          >
            {msg.content}
            {msg.operations && msg.operations.length > 0 && (
              <div className="mt-1 text-[10px] text-gray-500">
                {msg.operations.length} graph operation(s)
              </div>
            )}
          </div>
        ))}
        <div ref={messagesEndRef} />
      </div>

      <div className="border-t border-gray-700 p-2">
        <div className="flex gap-2">
          <input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSubmit()}
            placeholder="Describe your intent..."
            disabled={processing}
            className="flex-1 bg-gray-800 border border-gray-600 rounded px-3 py-2 text-xs focus:border-purple-500 focus:outline-none disabled:opacity-50"
          />
          <button
            onClick={handleSubmit}
            disabled={!input.trim() || processing}
            className="px-3 py-2 text-xs bg-purple-600 hover:bg-purple-500 text-white rounded disabled:opacity-50"
          >
            {processing ? "..." : "→"}
          </button>
        </div>
      </div>
    </div>
  );
}
