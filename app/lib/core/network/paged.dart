/// A single page of a paginated list plus the server's total count, so callers
/// can decide whether more pages remain (`items.length < total` across pages).
class Paged<T> {
  const Paged({required this.items, required this.total});

  final List<T> items;
  final int total;

  /// Maps the items to another type, preserving [total] (used to turn a page of
  /// DTOs into a page of domain entities).
  Paged<R> map<R>(R Function(T) f) =>
      Paged(items: items.map(f).toList(), total: total);
}
