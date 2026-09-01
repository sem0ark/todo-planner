export const Color = {
  deepVoid: "#001a24",
  baseVoid: "#003448",
  primaryText: "#ffffff",
  mutedText: "#dee2ef",
  secondaryText: "#91a6be",
  structuralBorder: "#afb6cf",
  offsetGreen: "#10b981",
  dashedBorder: "#dee2ef",
} as const;
export const Opacity = {
  subtleLine: 0.05,
  overlay: 0.1,
  border: 0.2,
  hover: 0.3,
  shadow: 0.5,
  icon: 0.4,
  progressBg: 0.2,
  progressFill: 0.6,
  label: 0.7,
  breatheMin: 0.65,
} as const;
export const Font = {
  ui: "Inter_400Regular",
  uiBold: "Inter_700Bold",
  uiBlack: "Inter_900Black",
  data: "JetBrainsMono_400Regular",
  dataBold: "JetBrainsMono_700Bold",
} as const;
export const Size = {
  categoryLarge: 28,
  body: 16,
  label: 12,
  mono: 12,
  categoryRow: 14,
} as const;
export const Space = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24 } as const;
export const Touch = { categoryRow: 56 } as const;

export function complementaryColor(hex: string): string {
  const normalized = hex.replace("#", "");
  if (!/^[0-9a-fA-F]{6}$/.test(normalized)) return Color.primaryText;
  const value = Number.parseInt(normalized, 16);
  return `#${(0xffffff ^ value).toString(16).padStart(6, "0")}`;
}
