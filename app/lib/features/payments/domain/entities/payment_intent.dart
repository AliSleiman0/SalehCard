/// Lifecycle of an on-chain USDT payment intent (mirrors the backend
/// `IntentStatus`). `confirming` = the transfer was seen on-chain and is being
/// settled; `confirmed` = money applied (wallet credited or order fulfilled).
enum PaymentIntentStatus { pending, confirming, confirmed, expired, unknown }

PaymentIntentStatus paymentIntentStatusFromString(String value) {
  switch (value) {
    case 'pending':
      return PaymentIntentStatus.pending;
    case 'confirming':
      return PaymentIntentStatus.confirming;
    case 'confirmed':
      return PaymentIntentStatus.confirmed;
    case 'expired':
      return PaymentIntentStatus.expired;
    default:
      return PaymentIntentStatus.unknown;
  }
}

/// Whether an on-chain USDT payment funds the wallet or settles an order.
enum PaymentPurpose { topup, order, unknown }

PaymentPurpose paymentPurposeFromString(String value) {
  switch (value) {
    case 'topup':
      return PaymentPurpose.topup;
    case 'order':
      return PaymentPurpose.order;
    default:
      return PaymentPurpose.unknown;
  }
}

/// A single on-chain USDT (TRC20) payment: a unique deposit address, the exact
/// amount to send, and the settlement state the app polls to completion.
class PaymentIntent {
  const PaymentIntent({
    required this.id,
    required this.purpose,
    required this.network,
    required this.address,
    required this.amountUsd,
    required this.status,
    this.orderId,
    this.receivedUsd = 0,
    this.txHash = '',
    this.createdAt,
    this.expiresAt,
  });

  final String id;
  final PaymentPurpose purpose;
  final String? orderId;
  final String network; // "trc20"
  final String address;
  final double amountUsd;
  final double receivedUsd;
  final PaymentIntentStatus status;
  final String txHash;
  final DateTime? createdAt;
  final DateTime? expiresAt;

  bool get isTerminal =>
      status == PaymentIntentStatus.confirmed ||
      status == PaymentIntentStatus.expired;

  bool get isConfirmed => status == PaymentIntentStatus.confirmed;
  bool get isExpired => status == PaymentIntentStatus.expired;
}

/// One admin-defined currency rate shown on the top-up screen (free text).
class ExchangeRate {
  const ExchangeRate({required this.label, required this.value});

  final String label;
  final String value;
}

/// The app-facing USDT feature gate (`GET /payments/config`).
class PaymentConfig {
  const PaymentConfig({
    required this.usdtEnabled,
    this.network = 'trc20',
    this.networks = const ['trc20'],
    this.expiryMinutes = 30,
    this.exchangeRates = const [],
  });

  final bool usdtEnabled;

  /// The default network (what an intent gets when none is picked).
  final String network;

  /// Every enabled network; a picker is shown only when there is more than one.
  final List<String> networks;

  final int expiryMinutes;

  /// Admin-maintained currency rates shown on the top-up screen (may be empty).
  final List<ExchangeRate> exchangeRates;
}
