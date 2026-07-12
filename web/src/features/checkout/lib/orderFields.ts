import type { CartItem } from '@/stores/cart'
import type { InputField, PlaceOrderItemInput, Product, Recipient } from '@/types'

// Non-quantity fields are the customer-editable delivery inputs; the legacy
// `quantity` type is a read-only echo of the cart qty (server derives it).
export function nonQuantityFields(product?: Product): InputField[] {
  return (product?.inputFields ?? []).filter((f) => f.type !== 'quantity')
}

// isValidLebaneseMobile mirrors the API's normalization (strip separators /
// international / trunk prefixes, re-expand the legacy 03 range) and the
// 03/70/71/76/78/79/81 + 6-digit prefix check. Bridge (recharge) products
// collect the target number in a `phone` field.
export function isValidLebaneseMobile(raw: string): boolean {
  let d = raw.replace(/[^0-9]/g, '')
  if (d.startsWith('00961')) d = d.slice(5)
  else if (d.startsWith('961')) d = d.slice(3)
  if (d.startsWith('0')) d = d.slice(1)
  if (d.length === 7 && d.startsWith('3')) d = '0' + d
  return /^(03|70|71|76|78|79|81)\d{6}$/.test(d)
}

// buildRecipient maps arbitrary transfer field values to {name, country, detail}
// by key-substring heuristic (port of checkout_screen.dart _buildRecipient).
// `values` is a fieldKey → value map for one line.
export function buildRecipient(values: Record<string, string>): Recipient {
  const pick = (needles: string[]): string => {
    for (const [k, v] of Object.entries(values)) {
      const lower = k.toLowerCase()
      if (needles.some((n) => lower.includes(n))) return v
    }
    return ''
  }
  const joined = Object.values(values).filter(Boolean).join(' / ')
  const name = pick(['name'])
  const detail = pick(['detail', 'account', 'number', 'iban', 'wallet', 'address'])
  return {
    name: name || joined,
    country: pick(['country']),
    detail: detail || joined,
  }
}

// buildOrderLine turns a cart line + its collected field values into a
// PlaceOrderItemInput (port of checkout_screen.dart _submit line-building).
// `itemValues` is a fieldKey → value map for this line. Transfer lines with no
// schema fields are handled by the caller (legacy name+country fallback).
export function buildOrderLine(
  item: CartItem,
  product: Product | undefined,
  itemValues: Record<string, string>,
): PlaceOrderItemInput {
  const fields = nonQuantityFields(product)
  const fulfillmentType = product?.fulfillmentType

  if (product && fulfillmentType !== 'code' && fields.length > 0) {
    const values: Record<string, string> = {}
    for (const f of fields) values[f.key] = (itemValues[f.key] ?? '').trim()

    if (fulfillmentType === 'transfer') {
      return {
        productId: item.id,
        variantId: item.variantId,
        qty: item.qty,
        recipient: buildRecipient(values),
      }
    }

    // account_credit: structured per-field capture, playerId = first filled field
    // (keeps the server's CreditedToID / api-mode fulfillment clean).
    const filled = fields
      .filter((f) => (values[f.key] ?? '').length > 0)
      .map((f) => ({ key: f.key, value: values[f.key] }))
    return {
      productId: item.id,
      variantId: item.variantId,
      qty: item.qty,
      playerId: filled.length ? filled[0].value : undefined,
      fields: filled.length ? filled : undefined,
    }
  }

  // code products / no schema fields: carry the pre-collected pid or recipient.
  return {
    productId: item.id,
    variantId: item.variantId,
    qty: item.qty,
    playerId: item.pid || undefined,
    recipient: item.recipient ?? undefined,
  }
}
