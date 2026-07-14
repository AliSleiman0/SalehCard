import { apiClient } from '@/lib/api-client'
import type { ApiResponse } from '@/types'

export interface I18nText {
  en: string
  ar: string
  tr: string
}

/** A node in the managed catalog taxonomy (Collection → Category → Subcategory). */
export interface AdminCategory {
  id: string
  parentId?: string // absent = a top-level Collection
  slug: string
  name: I18nText
  image?: string
  sortOrder: number
  rootDomain?: string
  depth: number // 0 = Collection, 1 = Category, 2 = Subcategory
  visible: boolean
  hasChildren: boolean
  productCount?: number
}

export interface CategoryCreateInput {
  parentId?: string // omit for a top-level Collection
  name: I18nText
  image?: string
  sortOrder?: number
  visible?: boolean
}

export interface CategoryUpdateInput {
  name?: I18nText
  image?: string
  sortOrder?: number
  visible?: boolean
  parentId?: string // move under a different parent
}

const ADMIN = '/api/admin/categories'

export function listCategories(): Promise<ApiResponse<AdminCategory[]>> {
  return apiClient.get<AdminCategory[]>(ADMIN)
}

export function createCategory(input: CategoryCreateInput): Promise<ApiResponse<AdminCategory>> {
  return apiClient.post<AdminCategory>(ADMIN, input)
}

export function updateCategory(id: string, input: CategoryUpdateInput): Promise<ApiResponse<AdminCategory>> {
  return apiClient.put<AdminCategory>(`${ADMIN}/${id}`, input)
}

export function deleteCategory(id: string): Promise<ApiResponse<{ deleted: boolean }>> {
  return apiClient.delete<{ deleted: boolean }>(`${ADMIN}/${id}`)
}

// --- tree helpers -----------------------------------------------------------

/** Nodes sorted into a stable pre-order (roots by sortOrder, children nested). */
export function orderedTree(nodes: AdminCategory[]): AdminCategory[] {
  const byParent = new Map<string, AdminCategory[]>()
  for (const n of nodes) {
    const key = n.parentId ?? ''
    const arr = byParent.get(key) ?? []
    arr.push(n)
    byParent.set(key, arr)
  }
  for (const arr of byParent.values()) {
    arr.sort((a, b) => a.sortOrder - b.sortOrder || a.name.en.localeCompare(b.name.en))
  }
  const out: AdminCategory[] = []
  const walk = (parentKey: string) => {
    for (const n of byParent.get(parentKey) ?? []) {
      out.push(n)
      walk(n.id)
    }
  }
  walk('')
  return out
}

/** "Games / PUBG / UC" — a node's English name path from its root. */
export function pathLabel(node: AdminCategory, byId: Map<string, AdminCategory>): string {
  const parts: string[] = [node.name.en || node.slug]
  let cur = node
  const seen = new Set<string>([node.id])
  while (cur.parentId) {
    const p = byId.get(cur.parentId)
    if (!p || seen.has(p.id)) break
    seen.add(p.id)
    parts.unshift(p.name.en || p.slug)
    cur = p
  }
  return parts.join(' / ')
}

/** IDs of a node and all its descendants — the parents a move must exclude. */
export function subtreeIds(rootId: string, nodes: AdminCategory[]): Set<string> {
  const byParent = new Map<string, AdminCategory[]>()
  for (const n of nodes) {
    const key = n.parentId ?? ''
    const arr = byParent.get(key) ?? []
    arr.push(n)
    byParent.set(key, arr)
  }
  const ids = new Set<string>([rootId])
  const walk = (id: string) => {
    for (const c of byParent.get(id) ?? []) {
      ids.add(c.id)
      walk(c.id)
    }
  }
  walk(rootId)
  return ids
}
