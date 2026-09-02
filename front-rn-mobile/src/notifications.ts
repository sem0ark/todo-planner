import {
  AndroidImportance,
  requestPermissionsAsync,
  scheduleNotificationAsync,
  setNotificationChannelAsync,
  setNotificationHandler,
} from "expo-notifications";
import { Platform } from "react-native";

export async function setupNotifications() {
  if (Platform.OS === "web") return false;

  setNotificationHandler({
    handleNotification: async () => ({
      shouldShowAlert: true,
      shouldPlaySound: true,
      shouldSetBadge: false,
      shouldShowBanner: false,
      shouldShowList: false,
    }),
  });
  const { status } = await requestPermissionsAsync();
  if (Platform.OS === "android")
    await setNotificationChannelAsync("boundaries", {
      name: "Block Boundaries",
      importance: AndroidImportance.HIGH,
    });
  return status === "granted";
}
export async function fireImmediateNotification(title: string, body: string) {
  if (Platform.OS === "web") return;

  await scheduleNotificationAsync({
    content: { title, body, sound: "default" },
    trigger: null,
  });
}
