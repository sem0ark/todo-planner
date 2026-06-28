import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useGetTodos } from "./hooks";
import { AddTodo } from "./AddTodo";
import { TodoItem } from "./TodoItem";

const queryClient = new QueryClient();

function TodoList() {
  const { data: todos, isLoading, error } = useGetTodos();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-gray-500">Loading todos...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
        Failed to load todos.
      </div>
    );
  }

  if (!todos || todos.length === 0) {
    return (
      <div className="text-center py-12 bg-gray-50 rounded-lg border border-gray-200">
        <p className="text-gray-500">
          No todos yet. Create one to get started!
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {todos.map((todo) => (
        <TodoItem key={todo.id} todo={todo} />
      ))}
    </div>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 py-8 px-4">
        <div className="max-w-2xl mx-auto">
          <header className="mb-8">
            <h1 className="text-4xl font-bold text-gray-900 mb-2">My Todos</h1>
            <p className="text-gray-600">Stay organized and productive</p>
          </header>

          <AddTodo />
          <TodoList />
        </div>
      </div>
    </QueryClientProvider>
  );
}

export default App;
