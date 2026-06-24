/// Authenticated user (domain entity).
class User {
  const User({
    required this.id,
    required this.email,
    required this.role,
    required this.locale,
    required this.walletBalance,
    required this.loyaltyPoints,
  });

  final String id;
  final String email;
  final String role;
  final String locale;
  final double walletBalance;
  final int loyaltyPoints;
}
