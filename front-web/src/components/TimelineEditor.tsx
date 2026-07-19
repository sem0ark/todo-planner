import { useState } from 'react';
import type { PlannedBlock } from '../services/templates';
import type { Category } from '../services/categories';

interface TimelineEditorProps {
  blocks: PlannedBlock[];
  categories: Category[];
  onChange: (blocks: PlannedBlock[]) => void;
}

export default function TimelineEditor({ blocks, categories, onChange }: TimelineEditorProps) {
  const [selectedBlock, setSelectedBlock] = useState<number | null>(null);

  const addBlock = () => {
    const lastBlock = blocks[blocks.length - 1];
    const newStartTime = lastBlock
      ? addMinutes(lastBlock.start_time, lastBlock.duration_minutes)
      : '00:00:00';

    const newBlock: PlannedBlock = {
      category_id: categories[0]?.id || 0,
      start_time: newStartTime,
      duration_minutes: 30,
    };

    onChange([...blocks, newBlock]);
  };

  const updateBlock = (index: number, updates: Partial<PlannedBlock>) => {
    const updated = blocks.map((block, i) =>
      i === index ? { ...block, ...updates } : block
    );
    onChange(updated);
  };

  const removeBlock = (index: number) => {
    onChange(blocks.filter((_, i) => i !== index));
  };

  const getCategoryColor = (categoryId: number) => {
    return categories.find((c) => c.id === categoryId)?.color || '#003448';
  };

  const getCategoryName = (categoryId: number) => {
    return categories.find((c) => c.id === categoryId)?.name || 'Unknown';
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-snow">Planned Blocks</h3>
        <button
          onClick={addBlock}
          className="px-3 py-1 text-sm font-semibold text-navy bg-snow rounded-lg transition-all duration-micro hover:bg-cloud"
          disabled={categories.length === 0}
        >
          Add Block
        </button>
      </div>

      {categories.length === 0 && (
        <p className="text-sm text-cloud">Create categories first before adding blocks.</p>
      )}

      <div className="space-y-2">
        {blocks.map((block, index) => (
          <div
            key={index}
            className={`p-4 bg-slate-blue/10 border-2 rounded-lg cursor-pointer transition-all duration-micro ${
              selectedBlock === index ? 'border-cloud' : 'border-slate-grey'
            }`}
            onClick={() => setSelectedBlock(selectedBlock === index ? null : index)}
          >
            <div className="flex items-center gap-4">
              <div
                className="w-6 h-6 rounded border-2 border-slate-grey"
                style={{ backgroundColor: getCategoryColor(block.category_id) }}
              />
              <div className="flex-1">
                <div className="text-snow font-medium">
                  {getCategoryName(block.category_id)}
                </div>
                <div className="text-sm text-cloud">
                  {formatTime(block.start_time)} · {block.duration_minutes} min
                </div>
              </div>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  removeBlock(index);
                }}
                className="px-2 py-1 text-sm text-error hover:text-error/80 transition-colors duration-micro"
              >
                Remove
              </button>
            </div>

            {selectedBlock === index && (
              <div className="mt-4 pt-4 border-t border-slate-grey space-y-3">
                <div>
                  <label className="block text-sm font-medium text-cloud mb-1">Category</label>
                  <select
                    value={block.category_id}
                    onChange={(e) => updateBlock(index, { category_id: parseInt(e.target.value) })}
                    className="w-full px-3 py-2 text-snow bg-navy/60 border-2 border-slate-grey rounded-lg outline-none transition-all duration-micro focus:border-cloud"
                  >
                    {categories.map((cat) => (
                      <option key={cat.id} value={cat.id}>
                        {cat.name}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-cloud mb-1">Start Time</label>
                  <input
                    type="time"
                    value={block.start_time.substring(0, 5)}
                    onChange={(e) => updateBlock(index, { start_time: e.target.value + ':00' })}
                    className="w-full px-3 py-2 text-snow bg-navy/60 border-2 border-slate-grey rounded-lg outline-none transition-all duration-micro focus:border-cloud"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-cloud mb-1">
                    Duration (minutes, min 30, step 15)
                  </label>
                  <input
                    type="number"
                    value={block.duration_minutes}
                    onChange={(e) => {
                      let value = parseInt(e.target.value) || 30;
                      value = Math.max(30, Math.round(value / 15) * 15);
                      updateBlock(index, { duration_minutes: value });
                    }}
                    min={30}
                    step={15}
                    className="w-full px-3 py-2 text-snow bg-navy/60 border-2 border-slate-grey rounded-lg outline-none transition-all duration-micro focus:border-cloud"
                  />
                </div>
              </div>
            )}
          </div>
        ))}

        {blocks.length === 0 && categories.length > 0 && (
          <p className="text-center text-cloud py-8">No blocks yet. Add blocks to build your template.</p>
        )}
      </div>
    </div>
  );
}

function formatTime(time: string): string {
  return time.substring(0, 5);
}

function addMinutes(time: string, minutes: number): string {
  const [hours, mins] = time.split(':').map(Number);
  const totalMinutes = hours * 60 + mins + minutes;
  const newHours = Math.floor(totalMinutes / 60) % 24;
  const newMins = totalMinutes % 60;
  return `${String(newHours).padStart(2, '0')}:${String(newMins).padStart(2, '0')}:00`;
}
