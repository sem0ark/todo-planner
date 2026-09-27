import React, { useMemo, useState } from "react";
import {
  DndContext,
  DragOverlay,
  useDraggable,
  useSensor,
  PointerSensor,
  type DragEndEvent,
  type DragStartEvent,
  type Modifier,
} from "@dnd-kit/core";
import { restrictToVerticalAxis } from "@dnd-kit/modifiers";
import { CSS } from "@dnd-kit/utilities";
import { constrainTimelineDragDelta } from "../utils/timeline";

type DragMode = "move" | "resize-top" | "resize-bottom";

export type LayoutItem = {
  id: string;
  offset: number; // Vertical position in grid units
  size: number; // Height in grid units
} & Record<string, any>; // Allow additional properties like color, etc.

export interface ItemPosition {
  width: number | string;
  left: number | string;
}

function calculateItemPositions(
  items: LayoutItem[],
  baseWidth: number | string,
): Record<string, ItemPosition> {
  const sorted = [...items].sort((a, b) => a.offset - b.offset);
  const columns: LayoutItem[][] = [];

  sorted.forEach((item) => {
    let placed = false;
    for (const col of columns) {
      const lastItem = col[col.length - 1];
      if (item.offset >= lastItem.offset + lastItem.size) {
        col.push(item);
        placed = true;
        break;
      }
    }
    if (!placed) columns.push([item]);
  });

  const positions: Record<string, ItemPosition> = {};
  columns.forEach((col, colIndex) => {
    col.forEach((item) => {
      const width =
        typeof baseWidth === "number"
          ? baseWidth / columns.length
          : 100 / columns.length;
      positions[item.id] = {
        width:
          typeof baseWidth === "number" ? width - 2 : `calc(${width}% - 2px)`,
        left:
          typeof baseWidth === "number"
            ? colIndex * width
            : `${colIndex * width}%`,
      };
    });
  });
  return positions;
}

interface DraggableColumnProps {
  items: LayoutItem[];
  gridUnit: number;
  baseWidth: number | string;
  onChange: (newItems: LayoutItem[]) => void;
  renderItem: (
    item: LayoutItem,
    elementType: "idle" | "dragging" | "overlay",
  ) => React.ReactNode;
  containerClassName?: string;
  itemClassName?: string;
  snapToInterval?: number; // Snap to multiples of this value (in grid units)
  minimumSize?: number;
  snapToEdges?: boolean;
  edgeSnapThreshold?: number;
  editable?: boolean;
}

function DragOverlayContent({
  item,
  position,
  gridUnit,
  renderItem,
  className,
}: {
  item: LayoutItem;
  position: ItemPosition;
  gridUnit: number;
  renderItem: DraggableColumnProps["renderItem"];
  dragMode: DragMode;
  className?: string;
}) {
  const style: React.CSSProperties = {
    width: position.width,
    height: item.size * gridUnit,
    opacity: 0.9,
    pointerEvents: "none",
  };

  return (
    <div style={style} className={`shadow-xl rounded-lg ${className}`}>
      {renderItem(item, "overlay")}
    </div>
  );
}

const DraggableItemWrapper = ({
  item,
  position,
  gridUnit,
  isOverlay = false,
  isDragging = false,
  renderItem,
  className,
  onDragModeChange,
  dragMode,
}: {
  item: LayoutItem;
  position: ItemPosition;
  gridUnit: number;
  isOverlay?: boolean;
  isDragging?: boolean;
  renderItem: DraggableColumnProps["renderItem"];
  className?: string;
  onDragModeChange?: (mode: DragMode) => void;
  dragMode: DragMode;
}) => {
  const { attributes, listeners, setNodeRef, transform } = useDraggable({
    id: item.id,
  });

  let visualTop = item.offset * gridUnit;
  let visualHeight = item.size * gridUnit;
  let visualTransform = CSS.Translate.toString(transform);

  if (transform && (isDragging || isOverlay)) {
    if (dragMode === "resize-bottom") {
      visualTransform = "none";
      visualHeight = Math.max(gridUnit, visualHeight + transform.y);
    } else if (dragMode === "resize-top") {
      visualHeight = Math.max(gridUnit, visualHeight - transform.y);
    }
  }

  const hasCompactResizeHandles = visualHeight < 40;

  const style: React.CSSProperties = {
    position: "absolute",
    top: visualTop,
    left: position.left,
    width: position.width,
    height: visualHeight,
    transform: visualTransform,
    zIndex: isDragging ? 50 : 10,
    touchAction: "none",
  };

  const handleResizeTopPointerDown = () => {
    onDragModeChange?.("resize-top");
  };

  const handleResizeBottomPointerDown = () => {
    onDragModeChange?.("resize-bottom");
  };

  const handleMovePointerDown = () => {
    onDragModeChange?.("move");
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`group cursor-grab active:cursor-grabbing ${className}`}
      {...attributes}
      {...listeners}
    >
      {/* Compact blocks move their resize handles outside the block so both remain usable. */}
      <div
        className={`absolute left-0 w-full h-5 cursor-ns-resize z-20 flex items-center justify-center transition-opacity ${hasCompactResizeHandles ? "-top-3 opacity-0 group-hover:opacity-100" : "top-0 opacity-0 group-hover:opacity-100"}`}
        onPointerDown={handleResizeTopPointerDown}
      >
        <div
          className={`flex gap-1 rounded-full px-2 py-1 ${hasCompactResizeHandles ? "bg-snow/90 shadow ring-1 ring-snow/70" : ""}`}
        >
          <div className="w-1 h-1 bg-slate-600 rounded-full" />
          <div className="w-1 h-1 bg-slate-600 rounded-full" />
          <div className="w-1 h-1 bg-slate-600 rounded-full" />
        </div>
      </div>

      {/* Main Content (Move Handle) */}
      <div className="w-full h-full" onPointerDown={handleMovePointerDown}>
        {renderItem(
          item,
          isOverlay ? "overlay" : isDragging ? "dragging" : "idle",
        )}
      </div>

      {/* Bottom Resize Handle (20px) */}
      <div
        className={`absolute left-0 w-full h-5 cursor-ns-resize z-20 flex items-center justify-center transition-opacity ${hasCompactResizeHandles ? "-bottom-3 opacity-0 group-hover:opacity-100" : "bottom-0 opacity-0 group-hover:opacity-100"}`}
        onPointerDown={handleResizeBottomPointerDown}
      >
        <div
          className={`flex gap-1 rounded-full px-2 py-1 ${hasCompactResizeHandles ? "bg-snow/90 shadow ring-1 ring-snow/70" : ""}`}
        >
          <div className="w-1 h-1 bg-slate-600 rounded-full" />
          <div className="w-1 h-1 bg-slate-600 rounded-full" />
          <div className="w-1 h-1 bg-slate-600 rounded-full" />
        </div>
      </div>
    </div>
  );
};

export function DraggableColumn({
  items,
  gridUnit,
  baseWidth,
  onChange,
  renderItem,
  containerClassName = "",
  itemClassName = "",
  snapToInterval = 1,
  minimumSize = snapToInterval,
  snapToEdges = false,
  edgeSnapThreshold = 8,
  editable = true,
}: DraggableColumnProps) {
  const [activeId, setActiveId] = useState<string | null>(null);
  const [dragMode, setDragMode] = useState<DragMode>("move");

  const sensors = [
    useSensor(PointerSensor, {
      activationConstraint: { distance: 2 },
    }),
  ];

  // Snap to grid modifier with interval
  const snapModifier: Modifier = ({ transform }) => {
    const snapSize = gridUnit * snapToInterval;
    let deltaUnits = Math.round(transform.y / snapSize) * snapToInterval;

    if (snapToEdges && activeId) {
      deltaUnits = constrainTimelineDragDelta(
        items,
        activeId,
        dragMode,
        deltaUnits,
        minimumSize,
        edgeSnapThreshold,
      );
    }

    return {
      ...transform,
      x: 0,
      y: deltaUnits * gridUnit,
    };
  };

  const handleDragStart = (event: DragStartEvent) => {
    setActiveId(event.active.id as string);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, delta } = event;
    setActiveId(null);

    const snapSize = gridUnit * snapToInterval;
    const deltaUnits = Math.round(delta.y / snapSize) * snapToInterval;

    const newItems = items.map((item) => {
      if (item.id !== active.id) return item;

      if (dragMode === "resize-top") {
        // Top resize: Offset moves down, and size changes inversely
        const newOffset = Math.max(0, item.offset + deltaUnits);
        const actualDelta = newOffset - item.offset;
        const newSize = Math.max(minimumSize, item.size - actualDelta);
        return { ...item, offset: newOffset, size: newSize };
      }

      if (dragMode === "resize-bottom") {
        // Bottom resize: Only size changes
        return {
          ...item,
          size: Math.max(minimumSize, item.size + deltaUnits),
        };
      }

      // Default: Move
      return { ...item, offset: Math.max(0, item.offset + deltaUnits) };
    });

    onChange(newItems);
    setDragMode("move"); // Reset to default mode
  };

  const itemPositions = useMemo(
    () => calculateItemPositions(items, baseWidth),
    [items, baseWidth],
  );

  const activeItem = items.find((i) => i.id === activeId);

  const handleDragModeChange = (mode: DragMode) => {
    setDragMode(mode);
  };

  return (
    <div
      className={`relative bg-app-void ${containerClassName}`}
      style={{
        width: baseWidth,
        height: "100%",
        backgroundImage: `linear-gradient(to bottom, #e5e7eb 1px, transparent 1px)`,
        backgroundSize: `100% ${gridUnit * 60}px`, // Grid line every hour
      }}
    >
      {editable ? (
        <DndContext
          sensors={sensors}
          modifiers={[restrictToVerticalAxis, snapModifier]}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
        >
          {items.map((item) => (
            <DraggableItemWrapper
              key={item.id}
              item={item}
              position={itemPositions[item.id]}
              gridUnit={gridUnit}
              isDragging={activeId === item.id}
              renderItem={renderItem}
              className={itemClassName}
              onDragModeChange={handleDragModeChange}
              dragMode={dragMode}
            />
          ))}

          <DragOverlay dropAnimation={null}>
            {activeItem ? (
              <DragOverlayContent
                item={activeItem}
                position={itemPositions[activeItem.id]}
                gridUnit={gridUnit}
                renderItem={renderItem}
                className={itemClassName}
                dragMode={dragMode}
              />
            ) : null}
          </DragOverlay>
        </DndContext>
      ) : (
        <>
          {items.map((item) => (
            <div
              key={item.id}
              className={itemClassName}
              style={{
                position: "absolute",
                top: item.offset * gridUnit,
                left: itemPositions[item.id].left,
                width: itemPositions[item.id].width,
                height: item.size * gridUnit,
              }}
            >
              {renderItem(item, "idle")}
            </div>
          ))}
        </>
      )}
    </div>
  );
}
