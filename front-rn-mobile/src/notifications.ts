import * as Notifications from "expo-notifications";
import { Platform } from "react-native";
export async function setupNotifications() {
  Notifications.setNotificationHandler({
    handleNotification: async () => ({
      shouldShowAlert: true,
      shouldPlaySound: true,
      shouldSetBadge: false,
    }),
  });
  const { status } = await Notifications.requestPermissionsAsync();
  if (Platform.OS === "android")
    await Notifications.setNotificationChannelAsync("boundaries", {
      name: "Block Boundaries",
      importance: Notifications.AndroidImportance.HIGH,
    });
  return status === "granted";
}
export async function fireImmediateNotification(title: string, body: string) {
  await Notifications.scheduleNotificationAsync({
    content: { title, body, sound: "default" },
    trigger: null,
  });
}
