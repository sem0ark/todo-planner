import React, { useState } from "react";
import {
  Pressable,
  StyleSheet,
  Text,
  View,
  useWindowDimensions,
} from "react-native";
import Svg, { Circle } from "react-native-svg";
import {
  Color,
  Font,
  Opacity,
  Size,
  Space,
  complementaryColor,
} from "../tokens";
import { useWidget } from "../WidgetContext";
import { parseTime, secondsOfDay } from "../time";

function formatRemaining(seconds: number) {
  const remaining = Math.max(0, Math.ceil(seconds));
  const minutes = Math.floor(remaining / 60);
  const secondsPart = String(remaining % 60).padStart(2, "0");
  return `${minutes}:${secondsPart}`;
}

function PomodoroRing({
  color,
  progress,
  size,
}: {
  color: string;
  progress: number;
  size: number;
}) {
  const strokeWidth = Math.max(4, size * 0.06);
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const dashOffset = circumference * (1 - Math.min(1, Math.max(0, progress)));
  return (
    <View style={{ width: size, height: size }}>
      <Svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
        <Circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          stroke="rgba(0,0,0,.2)"
          strokeWidth={strokeWidth}
          fill="none"
        />
        <Circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          stroke={color}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          strokeDasharray={`${circumference} ${circumference}`}
          strokeDashoffset={dashOffset}
          fill="none"
          rotation="-90"
          origin={`${size / 2}, ${size / 2}`}
        />
      </Svg>
    </View>
  );
}

export function Active() {
  const w = useWidget();
  const { width, height } = useWindowDimensions();
  const [blockWidth, setBlockWidth] = useState(0);
  const isHorizontal = width > height;
  if (!w.currentCategory)
    return (
      <View style={styles.empty}>
        <Text style={styles.emptyText}>SELECT A CATEGORY</Text>
        <Text style={styles.emptySubtext}>TAP A CATEGORY BELOW TO START</Text>
      </View>
    );
  const ringColor =
    w.state.pomPhase === "work"
      ? Color.primaryText
      : complementaryColor(w.currentCategory.color);
  const remainingSeconds = w.plannedBlock
    ? parseTime(w.plannedBlock.startTime) +
      w.plannedBlock.durationMinutes * 60 -
      secondsOfDay(Date.now())
    : 0;
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
        onLayout={(event) => setBlockWidth(event.nativeEvent.layout.width)}
        style={[
          styles.block,
          isHorizontal && styles.blockHorizontal,
          { backgroundColor: w.currentCategory.color },
        ]}
      >
        <Text style={styles.name}>{w.currentCategory.name.toUpperCase()}</Text>
        {!w.isOnSchedule && (
          <Text style={styles.expected}>
            EXPECTED: {w.plannedCategory?.name.toUpperCase()}
          </Text>
        )}
        <View style={styles.pomodoroHint}>
          {w.pomodoroActive && (
            <PomodoroRing
              color={ringColor}
              progress={w.pomodoroProgress}
              size={blockWidth / 3}
            />
          )}
        </View>
        <View style={styles.spacer} />
        <View style={styles.progressRow}>
          <View style={styles.track}>
            <View style={[styles.fill, { flex: w.progress }]} />
            <View style={{ flex: 1 - w.progress }} />
          </View>
          <Text style={styles.remaining}>
            {formatRemaining(remainingSeconds)}
          </Text>
        </View>
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1 },
  block: { flex: 1, width: "100%", padding: Space.lg },
  blockHorizontal: {},
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
  pomodoroHint: {
    ...StyleSheet.absoluteFillObject,
    alignItems: "center",
    justifyContent: "center",
  },
  progressRow: { flexDirection: "row", alignItems: "center", gap: Space.md },
  remaining: {
    color: Color.primaryText,
    fontFamily: Font.data,
    fontSize: Size.mono,
    opacity: Opacity.label,
    minWidth: 48,
    textAlign: "right",
  },
  spacer: { flex: 1 },
  track: {
    flex: 1,
    height: 5,
    flexDirection: "row",
    backgroundColor: "rgba(0,0,0,.2)",
  },
  fill: { backgroundColor: `rgba(255,255,255,${Opacity.progressFill})` },
  offset: {
    minHeight: 56,
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
    paddingHorizontal: Space.md,
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
    paddingHorizontal: Space.sm,
    paddingVertical: Space.sm,
    borderWidth: 1,
    borderColor: `rgba(16,185,129,${Opacity.icon})`,
    backgroundColor: Color.baseVoid,
  },
  empty: { flex: 1, alignItems: "center", justifyContent: "center" },
  emptyText: { color: Color.mutedText, fontFamily: Font.uiBold },
  emptySubtext: {
    color: Color.mutedText,
    fontFamily: Font.data,
    fontSize: Size.label,
    opacity: Opacity.label,
    marginTop: Space.sm,
  },
});
