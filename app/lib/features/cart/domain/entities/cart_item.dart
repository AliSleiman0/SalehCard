import '../../../../core/i18n/i18n_string.dart';

/// Recipient details for a money-transfer line item.
class Recipient {
  const Recipient({
    required this.name,
    required this.country,
    required this.detail,
  });

  final String name;
  final String country;
  final String detail;
}

/// A line item in the local cart. Mirrors `web/src/stores/cart.ts`: identified by
/// the composite `productId|variantId` key so the same denomination merges.
class CartItem {
  const CartItem({
    required this.productId,
    required this.variantId,
    required this.title,
    required this.category,
    required this.variantLabel,
    required this.price,
    required this.qty,
    required this.fulfillmentType, // code | account_credit | transfer
    this.playerId,
    this.recipient,
  });

  final String productId;
  final String variantId;
  final I18nString title;
  final String category;
  final String variantLabel;
  final double price;
  final int qty;
  final String fulfillmentType;
  final String? playerId;
  final Recipient? recipient;

  /// Composite key used for merge-on-add and removal.
  String get key => '$productId|$variantId';

  double get lineTotal => price * qty;

  CartItem copyWith({int? qty, String? playerId, Recipient? recipient}) {
    return CartItem(
      productId: productId,
      variantId: variantId,
      title: title,
      category: category,
      variantLabel: variantLabel,
      price: price,
      qty: qty ?? this.qty,
      fulfillmentType: fulfillmentType,
      playerId: playerId ?? this.playerId,
      recipient: recipient ?? this.recipient,
    );
  }
}
