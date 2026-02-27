import { useCallback, useState } from "react";

interface TourStep {
  target: string;
  title: string;
  content: string;
  position: "top" | "bottom" | "left" | "right";
}

const DEFAULT_STEPS: TourStep[] = [
  {
    target: "[data-tour='graph']",
    title: "Pipeline Graph",
    content: "Your pipeline visualized as an interactive graph. Drag nodes, click to inspect, scroll to zoom.",
    position: "bottom",
  },
  {
    target: "[data-tour='yaml-editor']",
    title: "YAML Editor",
    content: "Edit your pipeline YAML with LSP-powered completion, validation, and hover docs.",
    position: "left",
  },
  {
    target: "[data-tour='run-dashboard']",
    title: "Run Dashboard",
    content: "View past runs, launch new ones, and compare results side by side.",
    position: "left",
  },
  {
    target: "[data-tour='command-palette']",
    title: "Command Palette",
    content: "Press Ctrl+K to search nodes, actions, and recent runs.",
    position: "bottom",
  },
  {
    target: "[data-tour='overlays']",
    title: "Diagnostic Overlays",
    content: "Toggle heatmaps, walker traces, diffs, and more to analyze your pipeline.",
    position: "left",
  },
];

interface OnboardingTourProps {
  steps?: TourStep[];
  onComplete: () => void;
  onSkip: () => void;
}

export function OnboardingTour({
  steps = DEFAULT_STEPS,
  onComplete,
  onSkip,
}: OnboardingTourProps) {
  const [currentStep, setCurrentStep] = useState(0);

  const step = steps[currentStep];
  const isLast = currentStep === steps.length - 1;

  const next = useCallback(() => {
    if (isLast) {
      onComplete();
    } else {
      setCurrentStep((i) => i + 1);
    }
  }, [isLast, onComplete]);

  const prev = useCallback(() => {
    setCurrentStep((i) => Math.max(0, i - 1));
  }, []);

  return (
    <div className="fixed inset-0 z-[100] pointer-events-none">
      <div className="absolute inset-0 bg-black/40 pointer-events-auto" />

      <div
        className="absolute pointer-events-auto bg-gray-900 border border-gray-600 rounded-lg shadow-xl p-4 w-72"
        style={{
          left: "50%",
          top: "50%",
          transform: "translate(-50%, -50%)",
        }}
      >
        <div className="flex items-center justify-between mb-2">
          <span className="text-[10px] text-gray-500">
            {currentStep + 1} / {steps.length}
          </span>
          <button
            onClick={onSkip}
            className="text-[10px] text-gray-500 hover:text-white"
          >
            Skip tour
          </button>
        </div>

        <h3 className="text-sm font-semibold mb-1">{step.title}</h3>
        <p className="text-xs text-gray-400 mb-4">{step.content}</p>

        <div className="flex justify-between">
          <button
            onClick={prev}
            disabled={currentStep === 0}
            className="text-xs text-gray-500 hover:text-white disabled:opacity-30"
          >
            Back
          </button>
          <button
            onClick={next}
            className="text-xs bg-blue-600 hover:bg-blue-500 px-4 py-1.5 rounded"
          >
            {isLast ? "Finish" : "Next"}
          </button>
        </div>
      </div>
    </div>
  );
}
