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

/// A line item in the local cart. Identified by the composite
/// `productId|variantId|amount|playerId` key so the same denomination (or the
/// same typed amount of a per-unit product) for the same account merges, while
/// a purchase for a different player ID stays a separate line.
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
    this.originalPrice,
    this.playerId,
    this.recipient,
    this.amount,
  });

  final String productId;
  final String variantId;
  final I18nString title;
  final String category;
  final String variantLabel;

  /// Unit price actually charged — the discounted price when on sale. Drives
  /// [lineTotal] and the checkout subtotal (matches what the server charges).
  final double price;
  final int qty;
  final String fulfillmentType;

  /// Pre-discount unit price, kept for the struck-through display. Null (or equal
  /// to [price]) means no offer applies.
  final double? originalPrice;
  final String? playerId;
  final Recipient? recipient;

  /// Typed units on a per-unit (by-amount) line, e.g. 45000 coins; null for
  /// fixed-package lines. Such lines are always [qty] 1 and [price] is the
  /// whole line's price.
  final double? amount;

  /// Whether this is a per-unit (by-amount) line.
  bool get isAmountLine => amount != null;

  /// Composite key used for merge-on-add and removal. Includes the player ID
  /// (the first entered field) so two purchases for different accounts never
  /// collapse into one line.
  String get key => '$productId|$variantId|${amount ?? ''}|${playerId ?? ''}';

  double get lineTotal => price * qty;

  /// Whether this line carries a live discount below its base price.
  bool get hasOffer => originalPrice != null && originalPrice! > price;

  /// Pre-discount line total, for the struck-through display.
  double get originalLineTotal => (originalPrice ?? price) * qty;

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
      originalPrice: originalPrice,
      playerId: playerId ?? this.playerId,
      recipient: recipient ?? this.recipient,
      amount: amount,
    );
  }
}
