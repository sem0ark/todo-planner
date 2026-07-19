import { useState, useMemo, useEffect } from 'react';
import type { PlannedBlock } from '../services/templates';
import type { Category } from '../services/categories';
import { DraggableColumn, type LayoutItem } from './DraggableColumn';
import { getContrastTextColor } from '../utils/colors';
import { useAuthStore } from '../store/authStore';
import { useSettingsStore } from '../store/settingsStore';
import { getSettings } from '../services/settings';

const GRID_UNIT = 1; // 2px per minute
const BASE_WIDTH = 600;
const SNAP_INTERVAL = 15; // Snap to 15-minute intervals
const HOUR_HEIGHT = 60 * GRID_UNIT; // 60 minutes * 2px = 120px per hour

function timeToMinutes(time: string): number {
  const [hours, mins] = time.split(':').map(Number);
  return hours * 60 + mins;
}

function minutesToTime(minutes: number): string {
  const hours = Math.floor(minutes / 60) % 24;
  const mins = minutes % 60;
  return `${String(hours).padStart(2, '0')}:${String(mins).padStart(2, '0')}:00`;
}

function formatTime(time: string): string {
  return time.substring(0, 5);
}

export default function TimelineEditor({ blocks, categories, onChange }: {
  blocks: PlannedBlock[];
  categories: Category[];
  onChange: (blocks: PlannedBlock[]) => void;
}) {
  const { token } = useAuthStore();
  const { settings, setSettings } = useSettingsStore();
  const [selectedBlockId, setSelectedBlockId] = useState<string | null>(null);
  const [dayStartMinutes, setDayStartMinutes] = useState(0);

  useEffect(() => {
    if (token && !settings) {
      getSettings(token).then(setSettings).catch(console.error);
    }
  }, [token, settings, setSettings]);

  useEffect(() => {
    if (settings?.day_boundary_time) {
      setDayStartMinutes(timeToMinutes(settings.day_boundary_time));
    }
  }, [settings]);

  const layoutItems: LayoutItem[] = useMemo(() => {
    return blocks.map((block, index) => {
      const blockMinutes = timeToMinutes(block.start_time);
      // Offset relative to day start
      let offset = blockMinutes - dayStartMinutes;
      if (offset < 0) offset += 24 * 60; // Handle blocks that cross midnight

      return {
        id: `block-${index}`,
        offset,
        size: block.duration_minutes,
        categoryId: block.category_id,
        blockIndex: index,
      };
    });
  }, [blocks, dayStartMinutes]);

  const handleLayoutChange = (newItems: LayoutItem[]) => {
    const updatedBlocks = newItems.map((item) => {
      const originalBlock = blocks[item.blockIndex];
      let absoluteMinutes = dayStartMinutes + item.offset;
      if (absoluteMinutes >= 24 * 60) absoluteMinutes -= 24 * 60;

      return {
        ...originalBlock,
        start_time: minutesToTime(absoluteMinutes),
        duration_minutes: Math.max(15, Math.round(item.size / 15) * 15),
      };
    });
    onChange(updatedBlocks);
  };

  const addBlock = () => {
    const lastBlock = blocks[blocks.length - 1];
    const newStartMinutes = lastBlock
      ? timeToMinutes(lastBlock.start_time) + lastBlock.duration_minutes
      : dayStartMinutes;

    const newBlock: PlannedBlock = {
      category_id: categories[0]?.id || 0,
      start_time: minutesToTime(newStartMinutes % (24 * 60)),
      duration_minutes: 60,
    };

    onChange([...blocks, newBlock]);
  };

  const updateBlock = (index: number, updates: Partial<PlannedBlock>) => {
    const updated = blocks.map((block, i) =>
      i === index ? { ...block, ...updates } : block
    );
    onChange(updated);
  };

  const removeBlock = (blockId: string) => {
    const index = parseInt(blockId.split('-')[1]);
    onChange(blocks.filter((_, i) => i !== index));
    setSelectedBlockId(null);
  };

  const getCategoryColor = (categoryId: number) => {
    return categories.find((c) => c.id === categoryId)?.color || '#003448';
  };

  const getCategoryName = (categoryId: number) => {
    return categories.find((c) => c.id === categoryId)?.name || 'Unknown';
  };

  const selectedBlock = useMemo(() => {
    if (!selectedBlockId) return null;
    const index = parseInt(selectedBlockId.split('-')[1]);
    return blocks[index];
  }, [selectedBlockId, blocks]);

  const selectedBlockIndex = useMemo(() => {
    if (!selectedBlockId) return null;
    return parseInt(selectedBlockId.split('-')[1]);
  }, [selectedBlockId]);

  // Generate 24-hour time labels
  const timeLabels = useMemo(() => {
    const labels = [];
    for (let i = 0; i < 24; i++) {
      const hour = (dayStartMinutes / 60 + i) % 24;
      labels.push({
        hour: Math.floor(hour),
        offset: i * 60,
      });
    }
    return labels;
  }, [dayStartMinutes]);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-snow">Planned Blocks</h3>
        <button
          onClick={addBlock}
          className="px-4 py-2 text-sm font-semibold text-navy bg-snow rounded-lg transition-all duration-micro hover:bg-cloud"
          disabled={categories.length === 0}
        >
          Add Block
        </button>
      </div>

      {categories.length === 0 && (
        <p className="text-sm text-cloud">Create categories first before adding blocks.</p>
      )}

      <div className="flex gap-6">
        <div className="flex-1 bg-slate-blue/5 rounded-lg border border-slate-grey overflow-hidden py-2">
          <div className="flex">
            <div className="flex-shrink-0 w-16 bg-navy/40 border-slate-grey">
              {timeLabels.map(({ hour }) => (
                <div
                  key={hour}
                  className="text-xs text-cloud text-right pr-2 border-b border-slate-grey/30"
                  style={{ height: HOUR_HEIGHT }}
                >
                  <div className="relative -top-2">
                    {String(hour).padStart(2, '0')}:00
                  </div>
                </div>
              ))}
            </div>

            {/* Calendar column */}
            <div className="flex-1 relative">
              <DraggableColumn
                items={layoutItems}
                gridUnit={GRID_UNIT}
                baseWidth={BASE_WIDTH}
                snapToInterval={SNAP_INTERVAL}
                containerClassName="border-0 rounded-md"
                itemClassName=""
                onChange={handleLayoutChange}
                renderItem={(item, status) => {
                  const categoryColor = getCategoryColor(item.categoryId);
                  const categoryName = getCategoryName(item.categoryId);
                  const textColor = getContrastTextColor(categoryColor);
                  const blockId = item.id;
                  const isSelected = selectedBlockId === blockId;
                  const block = blocks[item.blockIndex];

                  return (
                    <div
                      className={
                        status === "overlay"
                          ? "hidden"
                          : `w-full h-full px-2 py-1 text-xs font-medium rounded shadow-sm cursor-pointer transition-shadow ${
                              status === "dragging" ? 'shadow-lg' : ''
                            } ${
                              isSelected ? 'ring-2 ring-yellow-400' : ''
                            }`
                      }
                      style={{
                        backgroundColor: categoryColor,
                        color: textColor,
                      }}
                      onClick={(e) => {
                        e.stopPropagation();
                        setSelectedBlockId(isSelected ? null : blockId);
                      }}
                    >
                      <div className="font-semibold truncate">{categoryName}</div>
                      <div className="text-[10px] opacity-80">
                        {formatTime(block.start_time)}, {item.size} min
                      </div>
                    </div>
                  );
                }}
              />
            </div>
          </div>
        </div>

        {/* Edit Panel */}
        <div className="w-80 flex-shrink-0">
          {selectedBlock && selectedBlockIndex !== null ? (
            <div className="p-4 bg-slate-blue/10 border border-slate-grey rounded-lg space-y-4">
              <div className="flex items-center justify-between">
                <h4 className="text-md font-semibold text-snow">Edit Block</h4>
                <button
                  onClick={() => removeBlock(selectedBlockId!)}
                  className="px-3 py-1 text-sm text-error hover:text-error/80 transition-colors duration-micro"
                >
                  Delete
                </button>
              </div>

              <div>
                <label className="block text-sm font-medium text-cloud mb-2">Category</label>
                <select
                  value={selectedBlock.category_id}
                  onChange={(e) => updateBlock(selectedBlockIndex, { category_id: parseInt(e.target.value) })}
                  className="w-full px-3 py-2 text-snow bg-navy/60 border border-slate-grey rounded-lg outline-none transition-all duration-micro focus:border-cloud"
                >
                  {categories.map((cat) => (
                    <option key={cat.id} value={cat.id}>
                      {cat.name}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-cloud mb-2">Start Time</label>
                <input
                  type="time"
                  value={selectedBlock.start_time.substring(0, 5)}
                  onChange={(e) => updateBlock(selectedBlockIndex, { start_time: e.target.value + ':00' })}
                  className="w-full px-3 py-2 text-snow bg-navy/60 border border-slate-grey rounded-lg outline-none transition-all duration-micro focus:border-cloud"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-cloud mb-2">
                  Duration (minutes)
                </label>
                <input
                  type="number"
                  value={selectedBlock.duration_minutes}
                  onChange={(e) => {
                    let value = parseInt(e.target.value) || 15;
                    value = Math.max(15, Math.round(value / 15) * 15);
                    updateBlock(selectedBlockIndex, { duration_minutes: value });
                  }}
                  min={15}
                  step={15}
                  className="w-full px-3 py-2 text-snow bg-navy/60 border border-slate-grey rounded-lg outline-none transition-all duration-micro focus:border-cloud"
                />
              </div>
            </div>
          ) : (
            <div className="p-4 bg-slate-blue/10 border border-slate-grey rounded-lg h-full flex items-center justify-center">
              <p className="text-cloud text-sm text-center">
                {blocks.length === 0
                  ? 'No blocks yet.\nAdd blocks to build your template.'
                  : 'Click on a block to edit'}
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
