// Per-profile text-size and brightness controls for comfortable viewing.
// Applied as CSS custom properties consumed by index.css (--font-scale)
// and every font-size declaration across the page CSS.
import { createContext, useContext, useLayoutEffect, type ReactNode } from 'react';
import { useUsers } from '../users/UsersContext';

export const FONT_LEVELS = [
  { id: 0, scale: 1, name: 'Default' },
  { id: 1, scale: 1.15, name: 'Large' },
  { id: 2, scale: 1.3, name: 'Larger' },
  { id: 3, scale: 1.45, name: 'Largest' },
] as const;

export const BRIGHTNESS_LEVELS = [
  { id: 0, value: 1, name: 'Bright (Default)' },
  { id: 1, value: 0.85, name: 'Normal' },
  { id: 2, value: 0.7, name: 'Dim' },
  { id: 3, value: 0.55, name: 'Dimmer' },
] as const;

interface DisplaySettingsValue {
  fontLevel: number;
  setFontLevel: (level: number) => void;
  brightnessLevel: number;
  setBrightnessLevel: (level: number) => void;
}

const DisplaySettingsContext = createContext<DisplaySettingsValue>({
  fontLevel: 0,
  setFontLevel: () => {},
  brightnessLevel: 0,
  setBrightnessLevel: () => {},
});

// Guards against an out-of-range level from a since-shrunk level list.
function resolveLevel(stored: number | null, maxIndex: number): number {
  if (stored === null || stored < 0 || stored > maxIndex) return 0;
  return stored;
}

export function DisplaySettingsProvider({ children }: { children: ReactNode }) {
  const { currentUser, updateUser } = useUsers();
  const fontLevel = resolveLevel(currentUser.fontLevel, FONT_LEVELS.length - 1);
  const brightnessLevel = resolveLevel(currentUser.brightnessLevel, BRIGHTNESS_LEVELS.length - 1);

  useLayoutEffect(() => {
    const scale = FONT_LEVELS[fontLevel]?.scale ?? 1;
    document.documentElement.style.setProperty('--font-scale', String(scale));
  }, [fontLevel]);

  useLayoutEffect(() => {
    const value = BRIGHTNESS_LEVELS[brightnessLevel]?.value ?? 1;
    document.documentElement.style.setProperty('--page-brightness', String(value));
  }, [brightnessLevel]);

  // A failed save reloads the server's value, so the UI reverts on its own.
  const setFontLevel = (level: number) => {
    updateUser(currentUser.id, { fontLevel: level }).catch(() => undefined);
  };
  const setBrightnessLevel = (level: number) => {
    updateUser(currentUser.id, { brightnessLevel: level }).catch(() => undefined);
  };

  return <DisplaySettingsContext.Provider value={{ fontLevel, setFontLevel, brightnessLevel, setBrightnessLevel }}>{children}</DisplaySettingsContext.Provider>;
}

export function useDisplaySettings() {
  return useContext(DisplaySettingsContext);
}
