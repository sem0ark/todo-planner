import React from "react";
import { Pressable, StyleSheet, Text, View } from "react-native";
import {
  Color,
  Font,
  Opacity,
  Size,
  Space,
  complementaryColor,
} from "../tokens";
import { useWidget } from "../WidgetContext";
export function Active() {
  const w = useWidget();
  if (!w.currentCategory)
    return (
      <View style={styles.empty}>
        <Text style={styles.emptyText}>SELECT A CATEGORY BELOW</Text>
      </View>
    );
  const ringColor =
    w.state.pomPhase === "work"
      ? Color.primaryText
      : complementaryColor(w.currentCategory.color);
  return (
    <View style={styles.root}>
      {!w.isOnSchedule && (
        <View style={styles.offset}>
          <Text style={styles.offsetText}>T-{w.state.offsetMinutes}M</Text>
          <Pressable onPress={() => w.offset(5)}>
            <Text style={styles.action}>+5M</Text>
          </Pressable>
          <Pressable onPress={w.returnToPlan}>
            <Text style={styles.action}>RETURN</Text>
          </Pressable>
        </View>
      )}
      <Pressable
        onPress={w.confirm}
        style={[styles.block, { backgroundColor: w.currentCategory.color }]}
      >
        <Text style={styles.name}>{w.currentCategory.name.toUpperCase()}</Text>
        {!w.isOnSchedule && (
          <Text style={styles.expected}>
            EXPECTED: {w.plannedCategory?.name.toUpperCase()}
          </Text>
        )}
        <View style={styles.pomodoroHint}>
          <Text style={{ color: ringColor }}>●</Text>
        </View>
        <View style={styles.spacer} />
        <View style={styles.track}>
          <View style={[styles.fill, { flex: w.progress }]} />
          <View style={{ flex: 1 - w.progress }} />
        </View>
      </Pressable>
    </View>
  );
}
const styles = StyleSheet.create({
  root: { flex: 1 },
  block: { flex: 1, padding: Space.xl },
  name: {
    fontFamily: Font.uiBlack,
    fontSize: Size.categoryLarge,
    color: Color.primaryText,
  },
  expected: {
    fontFamily: Font.uiBold,
    fontSize: Size.label,
    color: Color.primaryText,
    opacity: Opacity.label,
    marginTop: 4,
  },
  pomodoroHint: { alignItems: "center", paddingVertical: Space.lg },
  spacer: { flex: 1 },
  track: { height: 5, flexDirection: "row", backgroundColor: "rgba(0,0,0,.2)" },
  fill: { backgroundColor: `rgba(255,255,255,${Opacity.progressFill})` },
  offset: {
    height: 48,
    flexDirection: "row",
    alignItems: "center",
    gap: 16,
    paddingHorizontal: 12,
    backgroundColor: `rgba(0,52,72,${Opacity.overlay})`,
    borderBottomWidth: 1,
    borderBottomColor: `rgba(175,182,207,${Opacity.subtleLine})`,
  },
  offsetText: { color: Color.offsetGreen, fontFamily: Font.dataBold },
  action: {
    color: Color.primaryText,
    fontFamily: Font.data,
    fontSize: 12,
    opacity: Opacity.label,
  },
  empty: { flex: 1, alignItems: "center", justifyContent: "center" },
  emptyText: { color: Color.mutedText, fontFamily: Font.uiBold },
});
