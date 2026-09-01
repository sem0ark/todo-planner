import React from "react";
import {
  Pressable,
  SafeAreaView,
  StatusBar,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { Color, Font, Opacity, Size, Space } from "../tokens";
import { useWidget } from "../WidgetContext";
import { CategoryList } from "./CategoryList";
import { Active } from "./Active";
export function Shell() {
  const { state, plannedCategory, confirm } = useWidget();
  return (
    <SafeAreaView style={styles.safe}>
      <StatusBar barStyle="light-content" backgroundColor={Color.baseVoid} />
      <View style={styles.root}>
        <View style={styles.top}>
          {state.phase === "prompted" ? (
            <Pressable
              style={[
                styles.prompt,
                {
                  backgroundColor:
                    plannedCategory?.color ?? Color.structuralBorder,
                },
              ]}
              onPress={confirm}
            >
              <Text style={styles.name}>
                {plannedCategory?.name.toUpperCase()}
              </Text>
              <Text style={styles.hint}>TAP TO CONFIRM</Text>
            </Pressable>
          ) : state.phase === "active" ? (
            <Active />
          ) : null}
        </View>
        <View style={styles.divider} />
        <CategoryList />
      </View>
    </SafeAreaView>
  );
}
const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: Color.deepVoid },
  root: { flex: 1, backgroundColor: Color.baseVoid },
  top: { flex: 3, minHeight: 200 },
  divider: {
    height: 1,
    backgroundColor: Color.structuralBorder,
    opacity: Opacity.overlay,
  },
  prompt: {
    flex: 1,
    padding: Space.xl,
    justifyContent: "space-between",
    opacity: 1,
  },
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
