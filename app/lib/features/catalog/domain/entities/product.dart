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

/// Purchase-time account-ID verification config (check_name). [app] is the
/// provider's game slug (e.g. `pubgm-global`); [provider] is the numeric
/// verification-provider id. Present only on products that require verification.
class Verification {
  const Verification({required this.provider, required this.app});

  final int provider;
  final String app;
}

/// Buy/sell exchange-rate board for a money-transfer (`transfer`) product. Rates
/// are quote-currency units per 1 [baseCurrency] unit (e.g. 89500 LBP per $1).
/// Both rates are optional — a product may quote only [buyRate], only [sellRate],
/// or both; a null/zero rate means "not offered". [minAmount]/[maxAmount] bound
/// the base amount (null/0 = unbounded) and are informational in the rate board.
class MoneyTransfer {
  const MoneyTransfer({
    required this.baseCurrency,
    required this.quoteCurrency,
    this.buyRate,
    this.sellRate,
    this.minAmount,
    this.maxAmount,
  });

  final String baseCurrency;
  final String quoteCurrency;
  final double? buyRate;
  final double? sellRate;
  final double? minAmount;
  final double? maxAmount;

  /// Whether a usable buy rate is on offer.
  bool get hasBuy => (buyRate ?? 0) > 0;

  /// Whether a usable sell rate is on offer.
  bool get hasSell => (sellRate ?? 0) > 0;
}

/// Per-unit ("by amount") pricing for live-app recharge products (Up Live,
/// Bigo, Soul Star coins…): the customer types an amount into the
/// [fieldKey] input field and pays the single variant's effective price per
/// [blockUnits] units — see [unitLinePrice].
class UnitPricing {
  const UnitPricing({required this.fieldKey, required this.blockUnits});

  /// Key of the `amount`-type [InputField] that drives the price.
  final String fieldKey;

  /// How many units the single variant's price buys (e.g. 1000 coins).
  final double blockUnits;
}

/// Rounds a USD amount to the nearest cent, matching the API's `RoundCents`.
double roundCents(double v) => (v * 100).round() / 100;

/// Price of one per-unit line: the block price (already offer/reseller
/// adjusted) scaled to [amount] units, rounded to the cent — the same formula
/// the server charges (`blockPrice * amount / blockUnits`).
double unitLinePrice(double blockPrice, double amount, double blockUnits) {
  if (blockUnits <= 0) return 0;
  return roundCents(blockPrice * amount / blockUnits);
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
    this.thumbnail,
    this.fulfillmentType = 'code',
    this.inputFields = const [],
    this.verification,
    this.moneyTransfer,
    this.unitPricing,
    this.rating,
    this.ratingCount = 0,
    this.offer,
  });

  final String id;
  final I18nString title;
  final String category;
  final List<String> images;

  /// 256px compressed preview URL, or null. Lists/chips prefer it via
  /// [thumbUrl]; the full display image is [imageUrl].
  final String? thumbnail;

  final List<Variant> variants;
  final int stock;
  final bool available;

  /// Full-size display image URL (the 1024px image), or null when the product
  /// has no image. An empty string counts as no image (the catalog can store
  /// "" — see [thumbUrl]).
  String? get imageUrl {
    final first = images.isNotEmpty ? images.first : null;
    return (first != null && first.isNotEmpty) ? first : null;
  }

  /// Small preview URL for lists/chips: the thumbnail when present, else the
  /// full display image. Treats an empty-string [thumbnail] as absent — the
  /// backend stores "" for products without an uploaded thumbnail (e.g. every
  /// catalog-migration product), and "" must fall through to [imageUrl], not
  /// short-circuit it (a bare `thumbnail ?? imageUrl` would return "").
  String? get thumbUrl {
    final t = thumbnail;
    return (t != null && t.isNotEmpty) ? t : imageUrl;
  }

  /// Live sale on this product, or null when there is no active offer.
  final ProductOffer? offer;

  /// Whether a live offer applies to this product.
  bool get hasOffer => offer != null;

  /// What the customer receives: `code | account_credit | transfer`. Drives
  /// whether checkout collects dynamic [inputFields] (Player ID, etc).
  final String fulfillmentType;

  /// Dynamic per-product fields to collect at checkout (empty for most cards).
  final List<InputField> inputFields;

  /// Set when this product requires purchase-time ID verification; null
  /// otherwise.
  final Verification? verification;

  /// Whether the customer must verify a game ID (see nickname) before buying.
  bool get requiresIdVerification => verification != null;

  /// Buy/sell rate board for a money-transfer product; null otherwise.
  final MoneyTransfer? moneyTransfer;

  /// Whether to show the money-transfer rate board + calculator: a rate board
  /// is present and offers at least one rate.
  bool get isMoneyTransfer =>
      moneyTransfer != null &&
      (moneyTransfer!.hasBuy || moneyTransfer!.hasSell);

  /// Per-unit pricing spec, or null for normal (fixed-package) products.
  final UnitPricing? unitPricing;

  /// Whether the product is sold by a typed amount (coins etc.) rather than
  /// fixed packages. True even when misconfigured — check [unitAmountField]
  /// before allowing a purchase.
  bool get isUnitPriced => unitPricing != null;

  /// The `amount`-type input field that drives a per-unit product's price, or
  /// null when the product is not unit-priced or is misconfigured (block size
  /// not positive, driving field missing / not an amount, or not exactly one
  /// variant). A misconfigured unit-priced product must not be bought — the
  /// server would refuse it anyway.
  InputField? get unitAmountField {
    final u = unitPricing;
    if (u == null || u.blockUnits <= 0 || variants.length != 1) return null;
    for (final f in inputFields) {
      if (f.key == u.fieldKey && f.type == 'amount') return f;
    }
    return null;
  }

  /// Price of a unit-priced line for [amount] units, using the single
  /// variant's effective (offer/reseller) price, or null when the product is
  /// not validly unit-priced.
  double? unitLinePriceFor(double amount) {
    if (unitAmountField == null) return null;
    return unitLinePrice(
        variants.single.effectivePrice, amount, unitPricing!.blockUnits);
  }

  /// Units a unit-priced "from" price is quoted for: the driving field's
  /// minimum when set and positive, else one block.
  double? get _unitFromAmount {
    if (unitAmountField == null) return null;
    final min = unitAmountField!.constraints?.min;
    return (min != null && min > 0) ? min : unitPricing!.blockUnits;
  }

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
    final unitFrom = _unitFromAmount;
    if (unitFrom != null) {
      return unitLinePrice(
          variants.single.price, unitFrom, unitPricing!.blockUnits);
    }
    return variants
        .map((v) => v.price)
        .reduce((a, b) => a < b ? a : b);
  }

  /// Lowest discounted "from" price when on sale, else null. Prefers the
  /// server-computed [ProductOffer.offerFromPrice]; falls back to the lowest
  /// variant offer price.
  double? get offerFromPrice {
    final unitFrom = _unitFromAmount;
    if (unitFrom != null) {
      // The server's offer block quotes the block price — rescale it to the
      // minimum purchasable amount like [fromPrice].
      final p = variants.single.offerPrice;
      return p == null
          ? null
          : unitLinePrice(p, unitFrom, unitPricing!.blockUnits);
    }
    if (offer != null) return offer!.offerFromPrice;
    final priced = variants.where((v) => v.offerPrice != null);
    if (priced.isEmpty) return null;
    return priced.map((v) => v.offerPrice!).reduce((a, b) => a < b ? a : b);
  }
}
