/// Local date + time, e.g. "2026-11-12 15:05" — used everywhere a customer
/// sees when something happened (wallet rows, orders, notifications), so the
/// time is never dropped. Empty for null.
String formatDateTime(DateTime? at) {
  if (at == null) return '';
  final l = at.toLocal();
  String two(int v) => v.toString().padLeft(2, '0');
  return '${l.year}-${two(l.month)}-${two(l.day)} ${two(l.hour)}:${two(l.minute)}';
}
