import React from "react";
import { Pressable, ScrollView, StyleSheet, Text, View } from "react-native";
import { Color, Font, Space, Touch } from "../tokens";
import { useWidget } from "../WidgetContext";
export function CategoryList() {
  const { state, select } = useWidget();
  return (
    <ScrollView style={styles.root}>
      {state.categories.map((category, index) => (
        <Pressable
          key={category.id}
          onPress={() => select(category.id)}
          style={[
            styles.row,
            state.currentCategoryId === category.id && styles.active,
          ]}
        >
          <View
            style={[
              styles.dot,
              { backgroundColor: category.color },
              state.currentCategoryId === category.id && styles.activeDot,
            ]}
          />
          <Text style={styles.name}>{category.name.toUpperCase()}</Text>
          <Text style={styles.index}>{index + 1}</Text>
        </Pressable>
      ))}
    </ScrollView>
  );
}
const styles = StyleSheet.create({
  root: { flex: 2, backgroundColor: Color.baseVoid },
  row: {
    height: Touch.categoryRow,
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: Space.lg,
    borderBottomWidth: 1,
    borderBottomColor: "rgba(175,182,207,.05)",
  },
  active: { backgroundColor: "rgba(145,166,190,.2)" },
  dot: { width: 10, height: 10, borderRadius: 5, marginRight: Space.md },
  activeDot: { shadowColor: Color.primaryText, shadowOpacity: 0.8, shadowRadius: 4, elevation: 4 },
  name: {
    flex: 1,
    color: Color.secondaryText,
    fontFamily: Font.uiBold,
    fontSize: 14,
    letterSpacing: 1.2,
  },
  index: { color: Color.secondaryText, fontFamily: Font.data, fontSize: 12 },
});
