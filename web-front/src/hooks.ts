import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import API_URL from "./config";
import type { Todo, TodoInput } from "./types";

const getTodos = async (): Promise<Todo[]> => {
  const res = await fetch(`${API_URL}/todos`);
  if (!res.ok) throw new Error("Failed to fetch todos");
  return res.json();
};

const getTodo = async (id: number): Promise<Todo> => {
  const res = await fetch(`${API_URL}/todos/${id}`);
  if (!res.ok) throw new Error("Failed to fetch todo");
  return res.json();
};

const createTodo = async (todo: TodoInput): Promise<Todo> => {
  const res = await fetch(`${API_URL}/todos`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(todo),
  });
  if (!res.ok) throw new Error("Failed to create todo");
  return res.json();
};

const updateTodo = async (id: number, todo: TodoInput): Promise<Todo> => {
  const res = await fetch(`${API_URL}/todos/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(todo),
  });
  if (!res.ok) throw new Error("Failed to update todo");
  return res.json();
};

const deleteTodo = async (id: number): Promise<void> => {
  const res = await fetch(`${API_URL}/todos/${id}`, { method: "DELETE" });
  if (!res.ok) throw new Error("Failed to delete todo");
};

export const useGetTodos = () => {
  return useQuery({
    queryKey: ["todos"],
    queryFn: getTodos,
  });
};

export const useGetTodo = (id: number) => {
  return useQuery({
    queryKey: ["todos", id],
    queryFn: () => getTodo(id),
  });
};

export const useCreateTodo = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: createTodo,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["todos"] });
    },
  });
};

export const useUpdateTodo = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, todo }: { id: number; todo: TodoInput }) =>
      updateTodo(id, todo),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["todos"] });
    },
  });
};

export const useDeleteTodo = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: deleteTodo,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["todos"] });
    },
  });
};
