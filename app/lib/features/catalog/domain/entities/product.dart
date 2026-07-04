import '../../../../core/i18n/i18n_string.dart';

/// A purchasable denomination of a product (price is in USD).
class Variant {
  const Variant({
    required this.id,
    required this.denomination,
    required this.price,
    this.offerPrice,
  });

  final String id;
  final String denomination;
  final double price;

  /// Discounted unit price when a live offer applies to this product; null
  /// otherwise. Set by the catalog API's offer enrichment.
  final double? offerPrice;

  /// The price the customer actually pays: the offer price when on sale, else
  /// the base [price]. This is what flows into the cart and checkout.
  double get effectivePrice => offerPrice ?? price;

  /// Whether this variant carries a live discount below its base price.
  bool get hasOffer => offerPrice != null && offerPrice! < price;
}

/// A live sale-price deal attached to a catalog product. Mirrors the /offers
/// payload so the product screens can render the same struck-price + badge UI
/// the Offers tab uses.
class ProductOffer {
  const ProductOffer({
    required this.discountType,
    required this.discountValue,
    required this.originalFromPrice,
    required this.offerFromPrice,
    this.endsAt,
  });

  /// `percent` | `fixed`.
  final String discountType;
  final double discountValue;
  final double originalFromPrice;
  final double offerFromPrice;
  final DateTime? endsAt;

  /// Short badge label, e.g. "-25%" or "-$3" (matches the Offers tab).
  String get discountLabel {
    final v = discountValue == discountValue.roundToDouble()
        ? discountValue.toStringAsFixed(0)
        : discountValue.toStringAsFixed(2);
    return discountType == 'percent' ? '-$v%' : '-\$$v';
  }
}

/// Numeric ({min,max}) or enum ({options}) bounds for a dynamic input field.
class InputFieldConstraints {
  const InputFieldConstraints({this.min, this.max, this.options});

  final double? min;
  final double? max;
  final List<String>? options;
}

/// A customer-input spec entry collected at checkout (e.g. Player ID). Values of
/// [sensitive] fields must be masked in the UI. [type] is one of
/// `text | amount | quantity | select`.
class InputField {
  const InputField({
    required this.key,
    required this.label,
    required this.type,
    this.constraints,
    this.sensitive = false,
  });

  final String key;
  final I18nString label;
  final String type;
  final InputFieldConstraints? constraints;
  final bool sensitive;
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
    this.fulfillmentType = 'code',
    this.inputFields = const [],
    this.rating,
    this.ratingCount = 0,
    this.offer,
  });

  final String id;
  final I18nString title;
  final String category;
  final List<String> images;
  final List<Variant> variants;
  final int stock;
  final bool available;

  /// Live sale on this product, or null when there is no active offer.
  final ProductOffer? offer;

  /// Whether a live offer applies to this product.
  bool get hasOffer => offer != null;

  /// What the customer receives: `code | account_credit | transfer`. Drives
  /// whether checkout collects dynamic [inputFields] (Player ID, etc).
  final String fulfillmentType;

  /// Dynamic per-product fields to collect at checkout (empty for most cards).
  final List<InputField> inputFields;

  final double? rating;
  final int ratingCount;

  /// Orderable right now. `code` (inventory) fulfilment needs stock on hand;
  /// `account_credit` / `transfer` are fulfilled manually, so [available] alone
  /// suffices (the backend accepts these orders with stock 0).
  bool get inStock =>
      available && (fulfillmentType == 'code' ? stock > 0 : true);

  /// Lowest variant price, or null when there are no variants.
  double? get fromPrice {
    if (variants.isEmpty) return null;
    return variants
        .map((v) => v.price)
        .reduce((a, b) => a < b ? a : b);
  }

  /// Lowest discounted "from" price when on sale, else null. Prefers the
  /// server-computed [ProductOffer.offerFromPrice]; falls back to the lowest
  /// variant offer price.
  double? get offerFromPrice {
    if (offer != null) return offer!.offerFromPrice;
    final priced = variants.where((v) => v.offerPrice != null);
    if (priced.isEmpty) return null;
    return priced.map((v) => v.offerPrice!).reduce((a, b) => a < b ? a : b);
  }
}
