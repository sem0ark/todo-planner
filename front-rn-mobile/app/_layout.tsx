import React, { useEffect, useState } from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { Slot } from "expo-router";
import { useFonts } from "expo-font";
import {
  Inter_400Regular,
  Inter_700Bold,
  Inter_900Black,
} from "@expo-google-fonts/inter";
import {
  JetBrainsMono_400Regular,
  JetBrainsMono_700Bold,
} from "@expo-google-fonts/jetbrains-mono";
import { Color } from "../src/tokens";
import { WidgetProvider } from "../src/WidgetContext";
import { createMockRepository } from "../src/mock-repo";
import { setupNotifications } from "../src/notifications";
const repo = createMockRepository();
export default function RootLayout() {
  const [fontsLoaded] = useFonts({
    Inter_400Regular,
    Inter_700Bold,
    Inter_900Black,
    JetBrainsMono_400Regular,
    JetBrainsMono_700Bold,
  });
  const [ready, setReady] = useState(false);
  useEffect(() => {
    setupNotifications().then(() => setReady(true));
  }, []);
  if (!fontsLoaded || !ready)
    return (
      <View style={styles.loading}>
        <ActivityIndicator color={Color.primaryText} size="large" />
      </View>
    );
  return (
    <WidgetProvider repo={repo}>
      <Slot />
    </WidgetProvider>
  );
}
const styles = StyleSheet.create({
  loading: {
    flex: 1,
    backgroundColor: Color.baseVoid,
    justifyContent: "center",
    alignItems: "center",
  },
});
