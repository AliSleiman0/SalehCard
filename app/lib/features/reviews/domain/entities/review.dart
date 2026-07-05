/// The payload a customer submits to review a product. [body] is the optional
/// written note — the UI only collects it for low (1–2 star) ratings, and even
/// then it may be left blank, so an empty string is valid at any rating.
class ReviewDraft {
  const ReviewDraft({
    required this.productId,
    required this.rating,
    this.body = '',
  });

  final String productId;
  final int rating;
  final String body;
}

/// Whether the current user has already reviewed a product, and — when they
/// have — the moderation status of that review. Drives the product page's
/// "write a review" CTA state.
class MyReview {
  const MyReview({required this.reviewed, this.status});

  final bool reviewed;
  final String? status;

  /// The default "not reviewed yet" value (also the safe fallback on a failed
  /// lookup, so the CTA stays visible).
  static const none = MyReview(reviewed: false);
}
