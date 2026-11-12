import { apiClient } from '@/lib/api-client'
import type { ApiResponse } from '@/types'

/** One task on the shared admin to-do list. */
export interface Todo {
  id: string
  title: string
  note?: string
  dueAt?: string
  done: boolean
  doneAt?: string
  doneBy?: string
  createdBy?: string
  createdAt: string
}

export interface TodoInput {
  title?: string
  note?: string
  dueAt?: string
  clearDue?: boolean
  done?: boolean
}

const BASE = '/api/admin/todos'

export function listTodos(): Promise<ApiResponse<Todo[]>> {
  return apiClient.get<Todo[]>(BASE)
}
export function createTodo(input: TodoInput): Promise<ApiResponse<Todo>> {
  return apiClient.post<Todo>(BASE, input)
}
export function updateTodo(id: string, input: TodoInput): Promise<ApiResponse<Todo>> {
  return apiClient.put<Todo>(`${BASE}/${id}`, input)
}
export function deleteTodo(id: string): Promise<ApiResponse<{ deleted: boolean }>> {
  return apiClient.delete<{ deleted: boolean }>(`${BASE}/${id}`)
}
