import '../../../../core/i18n/i18n_string.dart';

/// A live sale-price deal on a catalog product (domain entity). The storefront
/// renders [originalFromPrice] struck through against [offerFromPrice].
class Offer {
  const Offer({
    required this.id,
    required this.productId,
    required this.productTitle,
    required this.images,
    required this.category,
    required this.inStock,
    required this.discountType,
    required this.discountValue,
    required this.originalFromPrice,
    required this.offerFromPrice,
    this.endsAt,
  });

  final String id;
  final String productId;
  final I18nString productTitle;
  final List<String> images;
  final String category;
  final bool inStock;

  /// `percent` | `fixed`.
  final String discountType;
  final double discountValue;
  final double originalFromPrice;
  final double offerFromPrice;
  final DateTime? endsAt;

  /// Short badge label, e.g. "-25%" or "-$3".
  String get discountLabel {
    final v = discountValue == discountValue.roundToDouble()
        ? discountValue.toStringAsFixed(0)
        : discountValue.toStringAsFixed(2);
    return discountType == 'percent' ? '-$v%' : '-\$$v';
  }
}
