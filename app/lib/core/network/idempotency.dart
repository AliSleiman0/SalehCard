import 'dart:math';

/// Generates a v4-style idempotency key for order submission. The backend dedupes
/// retries on the `Idempotency-Key` header (allowed by CORS); mint one per
/// checkout attempt and hold it across retries of the same attempt.
String newIdempotencyKey() {
  final rnd = Random();
  String hex(int n) =>
      List.generate(n, (_) => rnd.nextInt(16).toRadixString(16)).join();
  // 8-4-4-4-12 with version (4) and variant (8–b) nibbles.
  final variant = (8 + rnd.nextInt(4)).toRadixString(16);
  return '${hex(8)}-${hex(4)}-4${hex(3)}-$variant${hex(3)}-${hex(12)}';
}
