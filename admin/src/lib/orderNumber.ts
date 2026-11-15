/**
 * The order number customers and staff see everywhere (admin, web, app):
 * "#" + the last 8 characters of the order id, upper-cased — e.g. "#8D03458D".
 * Admin order search accepts it as typed (with or without the #).
 */
export function orderNumber(id: string): string {
  return `#${id.slice(-8).toUpperCase()}`
}
