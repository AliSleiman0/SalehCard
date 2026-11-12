import type { TopUpMethodField } from '../api/wallet'

/** Field key → submitted value (for `file` fields: the uploaded proof URL). */
export type MethodValues = Record<string, string>

/** Labels of required fields still empty (blocks submit). */
export function missingRequired(fields: TopUpMethodField[], values: MethodValues): string[] {
  return fields.filter((f) => f.required && !(values[f.key] ?? '').trim()).map((f) => f.label || f.key)
}

/** The request's `fields` payload: trimmed, empty values dropped, method order. */
export function methodFieldPayload(
  fields: TopUpMethodField[],
  values: MethodValues,
): { key: string; value: string }[] {
  return fields.map((f) => ({ key: f.key, value: (values[f.key] ?? '').trim() })).filter((f) => f.value)
}
