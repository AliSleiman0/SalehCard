import '../../../core/format/order_number.dart';
import '../../checkout/domain/entities/order.dart';

/// Photo field values (QR / barcode uploads) and any other link are never
/// searched: matching "https" or a blob host would light up unrelated orders.
final _url = RegExp(r'^https?://', caseSensitive: false);

String _fold(String s) => s.trim().toLowerCase();

/// Trims, lower-cases and drops leading `#`s, so "#A3F9C1", " a3f9c1 " and
/// "a3f9c1" all search the same thing.
String normalizeOrderQuery(String raw) =>
    _fold(raw).replaceFirst(RegExp(r'^#+'), '').trim();

/// The customer order-history search rule (client #79). Kept identical to the
/// web storefront's `matchesOrder` (web/src/features/orders/lib/matchOrder.ts):
///  - the order number shown on every row and order page (`#8D03458D`, see
///    [orderNumber]) or a prefix of it, and the full order id or a prefix of
///    it (the number is the id's last 8 characters, so a suffix of the id too);
///  - per line: player ID, checkout field values, transfer recipient name +
///    detail, and the product title in every locale.
/// Case-insensitive; an empty query matches everything. Values match as
/// substrings, the id only at its ends (a bare "12" would otherwise hit most
/// hex ids).
bool orderMatchesQuery(Order order, String rawQuery) {
  final q = normalizeOrderQuery(rawQuery);
  if (q.isEmpty) return true;

  final id = order.id.toLowerCase();
  final number = orderNumber(order.id).substring(1).toLowerCase();
  if (number.startsWith(q) || id.startsWith(q) || id.endsWith(q)) {
    return true;
  }

  bool hit(String? v) =>
      v != null && !_url.hasMatch(v.trim()) && _fold(v).contains(q);

  if (hit(order.fulfillment.transferRef)) return true;
  for (final item in order.items) {
    if (hit(item.playerId) ||
        hit(item.recipient?.name) ||
        hit(item.recipient?.detail)) {
      return true;
    }
    if (item.fields.any((f) => hit(f.value))) return true;
    if (item.title.values.values.any(hit)) return true;
  }
  return false;
}
