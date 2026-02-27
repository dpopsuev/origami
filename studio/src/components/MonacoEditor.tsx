import { useCallback, useEffect, useRef } from "react";

interface MonacoEditorProps {
  value: string;
  onChange: (value: string) => void;
  language?: string;
  readOnly?: boolean;
}

/**
 * Monaco-based YAML editor with LSP WebSocket connection.
 * Uses dynamic import to avoid bundling Monaco in the main chunk.
 */
export function MonacoEditor({
  value,
  onChange,
  language = "yaml",
  readOnly = false,
}: MonacoEditorProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<unknown>(null);
  const valueRef = useRef(value);

  useEffect(() => {
    valueRef.current = value;
  }, [value]);

  const initEditor = useCallback(async () => {
    if (!containerRef.current) return;

    // Dynamic import — Monaco loads on demand
    const monaco = await import("monaco-editor");

    const editor = monaco.editor.create(containerRef.current, {
      value,
      language,
      theme: "vs-dark",
      readOnly,
      minimap: { enabled: true },
      fontSize: 13,
      lineNumbers: "on",
      wordWrap: "on",
      scrollBeyondLastLine: false,
      automaticLayout: true,
      tabSize: 2,
      renderWhitespace: "selection",
      bracketPairColorization: { enabled: true },
    });

    editor.onDidChangeModelContent(() => {
      const newValue = editor.getValue();
      if (newValue !== valueRef.current) {
        valueRef.current = newValue;
        onChange(newValue);
      }
    });

    editorRef.current = editor;

    return () => {
      editor.dispose();
    };
  }, [language, readOnly]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    const cleanup = initEditor();
    return () => {
      cleanup?.then((fn) => fn?.());
    };
  }, [initEditor]);

  return (
    <div
      ref={containerRef}
      className="w-full h-full"
      data-testid="monaco-editor"
    />
  );
}
