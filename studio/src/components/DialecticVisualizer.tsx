interface DialecticRound {
  round: number;
  thesis: { walker: string; claim: string; confidence: number };
  antithesis: { walker: string; challenge: string; confidence: number };
  synthesis?: { verdict: string; confidence: number };
}

interface DialecticVisualizerProps {
  rounds: DialecticRound[];
  nodeName: string;
}

export function DialecticVisualizer({ rounds, nodeName }: DialecticVisualizerProps) {
  if (rounds.length === 0) {
    return (
      <div className="p-3 text-xs text-gray-500">
        No dialectic rounds for {nodeName}
      </div>
    );
  }

  return (
    <div className="p-3 space-y-3">
      <h3 className="text-xs font-semibold text-gray-400 uppercase tracking-wide">
        Dialectic: {nodeName}
      </h3>
      {rounds.map((round) => (
        <div key={round.round} className="border border-gray-700 rounded-lg overflow-hidden">
          <div className="bg-gray-800 px-3 py-1 text-xs text-gray-400">
            Round {round.round}
          </div>
          <div className="grid grid-cols-2 gap-0">
            <div className="p-2 border-r border-gray-700">
              <div className="text-[10px] text-blue-400 font-medium mb-1">
                Thesis ({round.thesis.walker})
              </div>
              <div className="text-xs">{round.thesis.claim}</div>
              <div className="text-[10px] text-gray-500 mt-1">
                confidence: {(round.thesis.confidence * 100).toFixed(0)}%
              </div>
            </div>
            <div className="p-2">
              <div className="text-[10px] text-red-400 font-medium mb-1">
                Antithesis ({round.antithesis.walker})
              </div>
              <div className="text-xs">{round.antithesis.challenge}</div>
              <div className="text-[10px] text-gray-500 mt-1">
                confidence: {(round.antithesis.confidence * 100).toFixed(0)}%
              </div>
            </div>
          </div>
          {round.synthesis && (
            <div className="px-3 py-2 bg-gray-800/50 border-t border-gray-700">
              <div className="text-[10px] text-green-400 font-medium">Synthesis</div>
              <div className="text-xs">{round.synthesis.verdict}</div>
              <div className="text-[10px] text-gray-500">
                confidence: {(round.synthesis.confidence * 100).toFixed(0)}%
              </div>
            </div>
          )}
        </div>
      ))}
    </div>
  );
}
