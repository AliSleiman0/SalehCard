import 'saved_player_id.dart';

/// Authenticated user (domain entity).
class User {
  const User({
    required this.id,
    required this.email,
    required this.role,
    required this.locale,
    required this.walletBalance,
    required this.loyaltyPoints,
    this.name = '',
    this.phone,
    this.savedPlayerIds = const [],
  });

  final String id;

  /// Display name captured at signup; empty for accounts created before names
  /// were collected.
  final String name;
  final String email;

  /// E.164 phone for phone-OTP accounts; null for email-only accounts.
  final String? phone;
  final String role;
  final String locale;
  final double walletBalance;
  final int loyaltyPoints;

  /// Player / account IDs the customer has saved for faster checkout. Persisted
  /// via `PATCH /users/me` (one of the only two editable profile fields).
  final List<SavedPlayerId> savedPlayerIds;
}
