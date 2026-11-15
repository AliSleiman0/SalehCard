import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Icon, PageHead, Modal, Checkbox, LoadingSpinner, ErrorState } from '@/components'
import type { TFunction } from 'i18next'
import { apiErrorText } from '@/lib/apiErrorText'
import { useRoles, usePermissionCatalog, useCreateRole, useUpdateRole, useDeleteRole } from '../hooks/useRoles'
import type { AdminRole, PermissionDomain } from '../api/roles'

/** A permission area's name in the admin's language; the server's English
 *  label is the fallback for an area added on the server later. */
function domainLabel(d: PermissionDomain, t: TFunction): string {
  return t(`role_domain_${d.key}`, { defaultValue: d.label })
}

// Super-admin-only page (guarded by the /roles route): create/edit/delete the
// custom RBAC roles that limited admin accounts are assigned. Permissions are a
// domains × {view, manage} grid served by the backend catalog; manage implies
// view (mirrors the server's NormalizePermissions). The built-in Super Admin is
// not listed — it is implicit (an admin with no role assigned) and immutable.
export default function RolesPage() {
  const { t } = useTranslation()
  const { data: rolesRes, isLoading, isError, refetch } = useRoles()
  const { data: catalogRes } = usePermissionCatalog()
  const del = useDeleteRole()

  const roles = rolesRes?.data ?? []
  const domains = catalogRes?.data?.domains ?? []

  // null = closed; 'new' = create; otherwise the role being edited.
  const [editing, setEditing] = useState<AdminRole | 'new' | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<AdminRole | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  function runDelete(role: AdminRole) {
    setDeleteError(null)
    del.mutate(role.id, {
      onSuccess: () => setConfirmDelete(null),
      onError: (e) => setDeleteError(apiErrorText(e, 'role_err_delete')),
    })
  }

  return (
    <div className="page">
      <PageHead
        back="/"
        crumbs={[t('grp_system'), t('nav_roles')]}
        title={t('nav_roles')}
        sub={t('role_sub')}
      >
        <button className="abtn primary" onClick={() => setEditing('new')}>
          <Icon name="plus" size={15} /> {t('role_new')}
        </button>
      </PageHead>

      <div className="acard">
        <div className="panelhead">
          <Icon name="shield" size={17} />
          <h3>{t('nav_roles')}</h3>
        </div>
        {isLoading && <LoadingSpinner />}
        {isError && <ErrorState message={t('role_load_error')} onRetry={() => void refetch()} />}
        {!isLoading && !isError && (
          <div className="tablewrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>{t('usr_col_role')}</th>
                  <th>{t('role_col_permissions')}</th>
                  <th style={{ width: 160 }}></th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td>
                    <b>{t('role_super_admin')}</b>
                    <div className="muted" style={{ fontSize: 12 }}>
                      {t('role_super_desc')}
                    </div>
                  </td>
                  <td>
                    <span className="bdg">{t('role_all_perms')}</span>
                  </td>
                  <td className="muted" style={{ fontSize: 12 }}>
                    {t('role_not_editable')}
                  </td>
                </tr>
                {roles.map((r) => (
                  <tr key={r.id}>
                    <td>
                      <b>{r.name}</b>
                      {r.description && (
                        <div className="muted" style={{ fontSize: 12 }}>{r.description}</div>
                      )}
                    </td>
                    <td>
                      <PermissionSummary role={r} domains={domains} />
                    </td>
                    <td>
                      <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
                        <button className="abtn sm" onClick={() => setEditing(r)}>
                          <Icon name="edit" size={14} /> {t('edit')}
                        </button>
                        <button
                          className="abtn sm danger"
                          onClick={() => {
                            setDeleteError(null)
                            setConfirmDelete(r)
                          }}
                        >
                          <Icon name="trash" size={14} />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
                {roles.length === 0 && (
                  <tr>
                    <td colSpan={3} className="muted" style={{ textAlign: 'center', padding: 20 }}>
                      {t('role_none')}
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        )}
        <div
          style={{
            padding: 16,
            borderTop: '1px solid var(--border)',
            display: 'flex',
            gap: 8,
            alignItems: 'center',
            color: 'var(--text-dim)',
            fontSize: 12.5,
          }}
        >
          <Icon name="alert" size={15} /> {t('role_footer')}
        </div>
      </div>

      {editing && (
        <RoleEditorModal
          role={editing === 'new' ? null : editing}
          domains={domains}
          onClose={() => setEditing(null)}
        />
      )}

      {confirmDelete && (
        <Modal onClose={() => setConfirmDelete(null)}>
          <div className="pad">
            <h3 style={{ marginTop: 0 }}>{t('role_delete_title', { name: confirmDelete.name })}</h3>
            <p className="muted" style={{ fontSize: 13, lineHeight: 1.6 }}>
              {t('role_delete_body')}
            </p>
            {deleteError && (
              <div style={{ color: 'var(--danger)', fontSize: 12.5, fontWeight: 600, marginBottom: 12 }}>
                {deleteError}
              </div>
            )}
            <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
              <button className="abtn" onClick={() => setConfirmDelete(null)}>
                {t('cancel')}
              </button>
              <button
                className="abtn danger"
                disabled={del.isPending}
                onClick={() => runDelete(confirmDelete)}
              >
                <Icon name="trash" size={14} /> {t('delete')}
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  )
}

/** Compact chips summarizing a role's domains: "Orders", "KYC ✎" (✎ = manage). */
function PermissionSummary({ role, domains }: { role: AdminRole; domains: PermissionDomain[] }) {
  const { t } = useTranslation()
  const parts = domains
    .filter((d) => role.permissions.includes(d.key + '.view'))
    .map((d) => ({
      label: domainLabel(d, t),
      manage: role.permissions.includes(d.key + '.manage'),
    }))
  if (parts.length === 0) return <span className="muted">—</span>
  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
      {parts.map((p) => (
        <span key={p.label} className="bdg" title={p.manage ? t('role_view_manage') : t('role_view_only')}>
          {p.label}
          {p.manage ? '' : ' 👁'}
        </span>
      ))}
    </div>
  )
}

/** Create/edit modal: name + description + the domains × view/manage grid. */
function RoleEditorModal({
  role,
  domains,
  onClose,
}: {
  role: AdminRole | null
  domains: PermissionDomain[]
  onClose: () => void
}) {
  const { t } = useTranslation()
  const create = useCreateRole()
  const update = useUpdateRole()
  const pending = create.isPending || update.isPending

  const [name, setName] = useState(role?.name ?? '')
  const [description, setDescription] = useState(role?.description ?? '')
  const [perms, setPerms] = useState<Set<string>>(new Set(role?.permissions ?? []))
  const [error, setError] = useState<string | null>(null)

  // manage implies view (matching the server's normalization); unchecking view
  // also drops manage so the grid can never show manage-without-view.
  function toggle(perm: string) {
    setPerms((prev) => {
      const next = new Set(prev)
      const [domain, kind] = perm.split('.')
      if (next.has(perm)) {
        next.delete(perm)
        if (kind === 'view') next.delete(domain + '.manage')
      } else {
        next.add(perm)
        if (kind === 'manage') next.add(domain + '.view')
      }
      return next
    })
  }

  function save() {
    setError(null)
    const trimmed = name.trim()
    if (!trimmed) {
      setError(t('role_err_name'))
      return
    }
    if (perms.size === 0) {
      setError(t('role_err_perm'))
      return
    }
    const input = { name: trimmed, description: description.trim(), permissions: [...perms] }
    const opts = {
      onSuccess: onClose,
      onError: (e: Error) => setError(apiErrorText(e, 'role_err_save')),
    }
    if (role) update.mutate({ id: role.id, input }, opts)
    else create.mutate(input, opts)
  }

  return (
    <Modal onClose={onClose} maxWidth={560}>
      <div className="pad">
        <h3 style={{ marginTop: 0 }}>{role ? t('role_edit_title', { name: role.name }) : t('role_new')}</h3>
        <div className="g2" style={{ marginBottom: 14 }}>
          <div>
            <label className="alabel">{t('res_name')}</label>
            <input
              className="afield"
              value={name}
              maxLength={60}
              placeholder={t('role_name_ph')}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div>
            <label className="alabel">{t('role_desc_label')}</label>
            <input
              className="afield"
              value={description}
              placeholder={t('role_desc_ph')}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
        </div>

        <label className="alabel">{t('role_perms_label')}</label>
        <div className="tablewrap" style={{ border: '1px solid var(--border)', borderRadius: 10, maxHeight: 320, overflowY: 'auto' }}>
          <table className="tbl">
            <thead>
              <tr>
                <th>{t('role_col_area')}</th>
                <th style={{ width: 80, textAlign: 'center' }}>{t('role_view')}</th>
                <th style={{ width: 80, textAlign: 'center' }}>{t('role_manage')}</th>
              </tr>
            </thead>
            <tbody>
              {domains.map((d) => (
                <tr key={d.key}>
                  <td style={{ fontWeight: 600, fontSize: 13 }}>{domainLabel(d, t)}</td>
                  <td style={{ textAlign: 'center' }}>
                    <div style={{ display: 'inline-flex' }}>
                      <Checkbox on={perms.has(d.key + '.view')} onClick={() => toggle(d.key + '.view')} />
                    </div>
                  </td>
                  <td style={{ textAlign: 'center' }}>
                    <div style={{ display: 'inline-flex' }}>
                      <Checkbox on={perms.has(d.key + '.manage')} onClick={() => toggle(d.key + '.manage')} />
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {error && (
          <div style={{ color: 'var(--danger)', fontSize: 12.5, fontWeight: 600, marginTop: 12 }}>
            {error}
          </div>
        )}
        <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 16 }}>
          <button className="abtn" onClick={onClose}>
            {t('cancel')}
          </button>
          <button className="abtn primary" disabled={pending} onClick={save}>
            <Icon name="check" size={15} /> {t('save')}
          </button>
        </div>
      </div>
    </Modal>
  )
}
