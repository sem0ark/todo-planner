import { useState } from "react";
import type { Todo } from "./types";
import { useUpdateTodo, useDeleteTodo } from "./hooks";

interface TodoItemProps {
  todo: Todo;
}

export function TodoItem({ todo }: TodoItemProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [editTitle, setEditTitle] = useState(todo.title);
  const updateTodo = useUpdateTodo();
  const deleteTodo = useDeleteTodo();

  const handleToggle = () => {
    updateTodo.mutate({
      id: todo.id,
      todo: {
        title: todo.title,
        description: todo.description,
        completed: !todo.completed,
        priority: todo.priority,
        due_date: todo.due_date,
      },
    });
  };

  const handleSave = () => {
    if (editTitle.trim()) {
      updateTodo.mutate({
        id: todo.id,
        todo: {
          title: editTitle,
          description: todo.description,
          completed: todo.completed,
          priority: todo.priority,
          due_date: todo.due_date,
        },
      });
      setIsEditing(false);
    }
  };

  const handleDelete = () => {
    if (confirm("Are you sure you want to delete this todo?")) {
      deleteTodo.mutate(todo.id);
    }
  };

  const priorityColor = {
    low: "bg-blue-100 text-blue-800",
    medium: "bg-yellow-100 text-yellow-800",
    high: "bg-red-100 text-red-800",
  };

  const dueDate = todo.due_date
    ? new Date(todo.due_date).toLocaleDateString()
    : null;

  return (
    <div
      className={`p-4 border rounded-lg ${todo.completed ? "bg-gray-50 border-gray-200" : "bg-white border-gray-300"} shadow-sm hover:shadow-md transition-shadow`}
    >
      <div className="flex items-start gap-3">
        <input
          type="checkbox"
          checked={todo.completed}
          onChange={handleToggle}
          className="w-5 h-5 mt-1 text-blue-600 rounded cursor-pointer"
        />
        <div className="flex-1">
          {isEditing ? (
            <div className="flex gap-2">
              <input
                type="text"
                value={editTitle}
                onChange={(e) => setEditTitle(e.target.value)}
                className="flex-1 px-2 py-1 border rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                autoFocus
              />
              <button
                onClick={handleSave}
                className="px-3 py-1 bg-blue-600 text-white rounded text-sm hover:bg-blue-700"
              >
                Save
              </button>
              <button
                onClick={() => {
                  setEditTitle(todo.title);
                  setIsEditing(false);
                }}
                className="px-3 py-1 bg-gray-300 text-gray-800 rounded text-sm hover:bg-gray-400"
              >
                Cancel
              </button>
            </div>
          ) : (
            <>
              <h3
                className={`font-medium ${todo.completed ? "line-through text-gray-500" : "text-gray-900"}`}
              >
                {todo.title}
              </h3>
              {todo.description && (
                <p
                  className={`text-sm mt-1 ${todo.completed ? "text-gray-400" : "text-gray-600"}`}
                >
                  {todo.description}
                </p>
              )}
            </>
          )}
          <div className="flex items-center gap-2 mt-2 flex-wrap">
            <span
              className={`text-xs px-2 py-1 rounded font-medium ${priorityColor[todo.priority]}`}
            >
              {todo.priority.charAt(0).toUpperCase() + todo.priority.slice(1)}
            </span>
            {dueDate && (
              <span className="text-xs text-gray-600 bg-gray-100 px-2 py-1 rounded">
                Due: {dueDate}
              </span>
            )}
          </div>
        </div>
        {!isEditing && (
          <div className="flex gap-2">
            <button
              onClick={() => setIsEditing(true)}
              className="px-3 py-1 text-sm text-blue-600 hover:bg-blue-50 rounded"
            >
              Edit
            </button>
            <button
              onClick={handleDelete}
              className="px-3 py-1 text-sm text-red-600 hover:bg-red-50 rounded"
            >
              Delete
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
