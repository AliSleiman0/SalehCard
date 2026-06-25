import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../domain/entities/cart_item.dart';

/// Local, in-memory cart (no backend until order submit). Items are keyed by
/// `productId|variantId`; adding an existing key merges quantity.
class CartController extends Notifier<List<CartItem>> {
  @override
  List<CartItem> build() => const [];

  void add(CartItem item) {
    final index = state.indexWhere((e) => e.key == item.key);
    if (index >= 0) {
      final existing = state[index];
      final next = [...state];
      next[index] = existing.copyWith(qty: existing.qty + item.qty);
      state = next;
    } else {
      state = [...state, item];
    }
  }

  void remove(String key) {
    state = state.where((e) => e.key != key).toList();
  }

  void setQty(String key, int qty) {
    if (qty <= 0) return remove(key);
    state = [
      for (final e in state) if (e.key == key) e.copyWith(qty: qty) else e,
    ];
  }

  void clear() => state = const [];
}

final cartControllerProvider =
    NotifierProvider<CartController, List<CartItem>>(CartController.new);

/// Total item count across the cart (for the bottom-nav badge).
final cartCountProvider = Provider<int>((ref) {
  return ref
      .watch(cartControllerProvider)
      .fold<int>(0, (sum, item) => sum + item.qty);
});

/// Cart subtotal in USD.
final cartSubtotalProvider = Provider<double>((ref) {
  return ref
      .watch(cartControllerProvider)
      .fold<double>(0, (sum, item) => sum + item.lineTotal);
});
