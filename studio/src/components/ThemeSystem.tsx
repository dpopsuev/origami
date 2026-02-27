import { createContext, useContext, useState, type ReactNode } from "react";

export interface StudioTheme {
  id: string;
  name: string;
  nodeShape: "rounded" | "sharp" | "pill";
  colorOverrides?: Record<string, string>;
  logo?: string;
  iconSet: "default" | "security" | "data" | "infra";
}

const DEFAULT_THEME: StudioTheme = {
  id: "rh-default",
  name: "Red Hat Default",
  nodeShape: "rounded",
  iconSet: "default",
};

interface ThemeContextValue {
  theme: StudioTheme;
  setTheme: (theme: StudioTheme) => void;
  darkMode: boolean;
  setDarkMode: (dark: boolean) => void;
}

const ThemeContext = createContext<ThemeContextValue>({
  theme: DEFAULT_THEME,
  setTheme: () => {},
  darkMode: true,
  setDarkMode: () => {},
});

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setTheme] = useState<StudioTheme>(DEFAULT_THEME);
  const [darkMode, setDarkMode] = useState(true);

  return (
    <ThemeContext.Provider value={{ theme, setTheme, darkMode, setDarkMode }}>
      <div className={darkMode ? "dark" : ""}>{children}</div>
    </ThemeContext.Provider>
  );
}

export function useStudioTheme() {
  return useContext(ThemeContext);
}

interface ThemeSelectorProps {
  themes: StudioTheme[];
}

export function ThemeSelector({ themes }: ThemeSelectorProps) {
  const { theme, setTheme, darkMode, setDarkMode } = useStudioTheme();

  return (
    <div className="p-3 space-y-3">
      <h3 className="text-xs font-semibold text-gray-400 uppercase tracking-wide">
        Theme
      </h3>

      <div className="space-y-1">
        {themes.map((t) => (
          <button
            key={t.id}
            onClick={() => setTheme(t)}
            className={`w-full text-left p-2 rounded text-xs ${
              theme.id === t.id
                ? "bg-blue-900/30 border border-blue-500/30"
                : "bg-gray-800 border border-gray-700 hover:border-gray-600"
            }`}
          >
            <div className="font-medium">{t.name}</div>
            <div className="text-[10px] text-gray-500">
              {t.nodeShape} nodes · {t.iconSet} icons
            </div>
          </button>
        ))}
      </div>

      <div className="flex items-center justify-between">
        <span className="text-xs text-gray-400">Dark mode</span>
        <button
          onClick={() => setDarkMode(!darkMode)}
          className={`w-10 h-5 rounded-full transition-colors ${
            darkMode ? "bg-blue-600" : "bg-gray-600"
          }`}
        >
          <div
            className={`w-4 h-4 bg-white rounded-full transition-transform mx-0.5 ${
              darkMode ? "translate-x-5" : "translate-x-0"
            }`}
          />
        </button>
      </div>
    </div>
  );
}
