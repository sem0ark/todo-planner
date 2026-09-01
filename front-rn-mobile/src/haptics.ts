import * as Haptics from "expo-haptics";
export async function haptic(
  pattern: "success" | "tap" | "prompt" | "pomodoro",
) {
  if (pattern === "prompt")
    return Haptics.notificationAsync(Haptics.NotificationFeedbackType.Warning);
  return Haptics.impactAsync(
    pattern === "pomodoro"
      ? Haptics.ImpactFeedbackStyle.Medium
      : Haptics.ImpactFeedbackStyle.Light,
  );
}
