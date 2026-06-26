/// Authenticated user (domain entity).
class User {
  const User({
    required this.id,
    required this.email,
    required this.role,
    required this.locale,
    required this.walletBalance,
    required this.loyaltyPoints,
    this.savedPlayerIds = const [],
  });

  final String id;
  final String email;
  final String role;
  final String locale;
  final double walletBalance;
  final int loyaltyPoints;

  /// Player / account IDs the customer has saved for faster checkout. Persisted
  /// via `PATCH /users/me` (one of the only two editable profile fields).
  final List<String> savedPlayerIds;
}
