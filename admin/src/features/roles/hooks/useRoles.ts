import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  listRoles,
  getPermissionCatalog,
  createRole,
  updateRole,
  deleteRole,
  type RoleInput,
} from '../api/roles'

export function useRoles() {
  return useQuery({ queryKey: ['admin', 'roles'], queryFn: () => listRoles() })
}

export function usePermissionCatalog() {
  return useQuery({
    queryKey: ['admin', 'roles', 'permissions'],
    queryFn: () => getPermissionCatalog(),
    staleTime: Infinity, // the catalog is code-defined, it never changes at runtime
  })
}

export function useCreateRole() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: RoleInput) => createRole(input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'roles'] })
    },
  })
}

export function useUpdateRole() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: RoleInput }) => updateRole(id, input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'roles'] })
    },
  })
}

export function useDeleteRole() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteRole(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'roles'] })
    },
  })
}
