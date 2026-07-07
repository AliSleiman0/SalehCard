/// Lifecycle of a payment intent (mirrors the backend `IntentStatus`).
/// `confirming` = the payment was seen (on-chain transfer / Whish callback) and
/// is being settled; `confirmed` = money applied (wallet credited or order
/// fulfilled); `failed` = a Whish gateway-reported failure (terminal).
enum PaymentIntentStatus {
  pending,
  confirming,
  confirmed,
  expired,
  failed,
  unknown,
}

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
    case 'failed':
      return PaymentIntentStatus.failed;
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

/// A single payment intent the app polls to completion. USDT intents carry a
/// deposit [address] (+ [network]); Whish intents carry a hosted [redirectUrl]
/// the app opens in the browser. [provider] is "usdt" or "whish".
class PaymentIntent {
  const PaymentIntent({
    required this.id,
    required this.purpose,
    required this.amountUsd,
    required this.status,
    this.provider = 'usdt',
    this.network = '',
    this.address = '',
    this.redirectUrl = '',
    this.orderId,
    this.receivedUsd = 0,
    this.txHash = '',
    this.createdAt,
    this.expiresAt,
  });

  final String id;
  final String provider; // "usdt" | "whish"
  final PaymentPurpose purpose;
  final String? orderId;
  final String network; // "trc20" (usdt)
  final String address; // deposit address (usdt)
  final String redirectUrl; // hosted collect URL (whish)
  final double amountUsd;
  final double receivedUsd;
  final PaymentIntentStatus status;
  final String txHash;
  final DateTime? createdAt;
  final DateTime? expiresAt;

  bool get isWhish => provider == 'whish';

  bool get isTerminal =>
      status == PaymentIntentStatus.confirmed ||
      status == PaymentIntentStatus.expired ||
      status == PaymentIntentStatus.failed;

  bool get isConfirmed => status == PaymentIntentStatus.confirmed;
  bool get isExpired => status == PaymentIntentStatus.expired;
  bool get isFailed => status == PaymentIntentStatus.failed;
}

/// The app-facing payment feature gate (`GET /payments/config`).
class PaymentConfig {
  const PaymentConfig({
    this.usdtEnabled = false,
    this.whishEnabled = false,
    this.network = 'trc20',
    this.expiryMinutes = 30,
  });

  final bool usdtEnabled;
  final bool whishEnabled;
  final String network;
  final int expiryMinutes;
}
