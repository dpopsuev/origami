import { ReactFlowProvider } from "@xyflow/react";
import { PipelineGraph } from "./components/PipelineGraph";

export function App() {
  return (
    <div className="h-screen w-screen flex flex-col bg-rh-dark text-white">
      <header className="h-12 flex items-center px-4 border-b border-gray-700 shrink-0">
        <h1 className="text-lg font-semibold">Origami Studio</h1>
      </header>
      <main className="flex-1 overflow-hidden">
        <ReactFlowProvider>
          <PipelineGraph />
        </ReactFlowProvider>
      </main>
    </div>
  );
}
