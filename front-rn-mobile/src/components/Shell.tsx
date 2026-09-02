import React, { useEffect, useRef } from "react";
import {
  Animated,
  Pressable,
  StatusBar,
  StyleSheet,
  Text,
  View,
  useWindowDimensions,
} from "react-native";
import { Color, Font, Opacity, Size, Space } from "../tokens";
import { useWidget } from "../WidgetContext";
import { CategoryList } from "./CategoryList";
import { Active } from "./Active";
import { SafeAreaView } from "react-native-safe-area-context";

export function Shell() {
  const { state, plannedCategory, confirm } = useWidget();
  const { width, height } = useWindowDimensions();
  const isHorizontal = width > height;
  return (
    <SafeAreaView style={styles.safe}>
      <StatusBar barStyle="light-content" backgroundColor={Color.baseVoid} />
      <View style={[styles.root, isHorizontal && styles.rootHorizontal]}>
        <View style={[styles.top, isHorizontal && styles.topHorizontal]}>
          {state.phase === "prompted" ? (
            <Prompt
              promptColor={plannedCategory?.color ?? Color.structuralBorder}
              name={plannedCategory?.name}
              onConfirm={confirm}
            />
          ) : state.phase === "active" ? (
            <Active />
          ) : null}
        </View>
        <View style={[styles.divider, isHorizontal && styles.dividerHorizontal]} />
        <CategoryList />
      </View>
    </SafeAreaView>
  );
}

function Prompt({
  promptColor,
  name,
  onConfirm,
}: {
  promptColor: string;
  name?: string;
  onConfirm: () => void;
}) {
  const opacity = useRef(new Animated.Value(1)).current;

  useEffect(() => {
    const animation = Animated.loop(
      Animated.sequence([
        Animated.timing(opacity, {
          toValue: Opacity.breatheMin,
          duration: 1000,
          useNativeDriver: true,
        }),
        Animated.timing(opacity, {
          toValue: 1,
          duration: 1000,
          useNativeDriver: true,
        }),
      ]),
    );
    animation.start();
    return () => animation.stop();
  }, [opacity]);

  return (
    <Pressable style={styles.prompt} onPress={onConfirm}>
      <Animated.View
        style={[styles.promptBackground, { backgroundColor: promptColor, opacity }]}
      />
      <View style={styles.promptFill}>
        <Text style={styles.name}>{name?.toUpperCase()}</Text>
        <Text style={styles.hint}>TAP TO CONFIRM</Text>
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: Color.deepVoid },
  root: { flex: 1, backgroundColor: Color.baseVoid },
  rootHorizontal: { flexDirection: "row" },
  top: { minHeight: 200, aspectRatio: 1 },
  topHorizontal: { height: "100%", aspectRatio: 1 },
  divider: {
    height: 1,
    backgroundColor: Color.structuralBorder,
    opacity: Opacity.overlay,
  },
  dividerHorizontal: { width: 1, height: "100%" },
  prompt: {
    flex: 1,
  },
  promptFill: {
    flex: 1,
    padding: Space.lg,
    justifyContent: "space-between",
  },
  promptBackground: { ...StyleSheet.absoluteFillObject },
  name: {
    fontFamily: Font.uiBlack,
    fontSize: Size.categoryLarge,
    color: Color.primaryText,
  },
  hint: {
    fontFamily: Font.dataBold,
    fontSize: Size.label,
    color: Color.primaryText,
    opacity: Opacity.label,
    letterSpacing: 2,
  },
});
