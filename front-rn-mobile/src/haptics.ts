import {
  impactAsync,
  ImpactFeedbackStyle,
  notificationAsync,
  NotificationFeedbackType,
} from "expo-haptics";

export async function haptic(
  pattern: "success" | "tap" | "prompt" | "pomodoro",
) {
  if (pattern === "prompt")
    return notificationAsync(NotificationFeedbackType.Warning);

  return impactAsync(
    pattern === "pomodoro"
      ? ImpactFeedbackStyle.Medium
      : ImpactFeedbackStyle.Light,
  );
}
