import '../../../../core/i18n/i18n_string.dart';

/// A purchasable denomination of a product (price is in USD).
class Variant {
  const Variant({
    required this.id,
    required this.denomination,
    required this.price,
  });

  final String id;
  final String denomination;
  final double price;
}

/// Catalog product (domain entity).
class Product {
  const Product({
    required this.id,
    required this.title,
    required this.category,
    required this.images,
    required this.variants,
    required this.stock,
    required this.available,
    this.rating,
    this.ratingCount = 0,
  });

  final String id;
  final I18nString title;
  final String category;
  final List<String> images;
  final List<Variant> variants;
  final int stock;
  final bool available;
  final double? rating;
  final int ratingCount;

  /// Lowest variant price, or null when there are no variants.
  double? get fromPrice {
    if (variants.isEmpty) return null;
    return variants
        .map((v) => v.price)
        .reduce((a, b) => a < b ? a : b);
  }
}
