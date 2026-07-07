import '../../domain/entities/payment_intent.dart';

/// Hand-written DTO (no build_runner) for a payment intent — mirrors the
/// backend `IntentView`.
class PaymentIntentDto {
  const PaymentIntentDto({
    required this.id,
    this.provider = 'usdt',
    this.purpose = '',
    this.orderId,
    this.network = '',
    this.address = '',
    this.redirectUrl = '',
    this.amountUsd = 0,
    this.receivedUsd = 0,
    this.status = '',
    this.txHash = '',
    this.createdAt,
    this.expiresAt,
  });

  final String id;
  final String provider;
  final String purpose;
  final String? orderId;
  final String network;
  final String address;
  final String redirectUrl;
  final double amountUsd;
  final double receivedUsd;
  final String status;
  final String txHash;
  final String? createdAt;
  final String? expiresAt;

  factory PaymentIntentDto.fromJson(Map<String, dynamic> json) =>
      PaymentIntentDto(
        id: json['id'] as String? ?? '',
        provider: json['provider'] as String? ?? 'usdt',
        purpose: json['purpose'] as String? ?? '',
        orderId: json['orderId'] as String?,
        network: json['network'] as String? ?? '',
        address: json['address'] as String? ?? '',
        redirectUrl: json['redirectUrl'] as String? ?? '',
        amountUsd: (json['amountUsd'] as num?)?.toDouble() ?? 0,
        receivedUsd: (json['receivedUsd'] as num?)?.toDouble() ?? 0,
        status: json['status'] as String? ?? '',
        txHash: json['txHash'] as String? ?? '',
        createdAt: json['createdAt'] as String?,
        expiresAt: json['expiresAt'] as String?,
      );

  PaymentIntent toEntity() => PaymentIntent(
        id: id,
        provider: provider,
        purpose: paymentPurposeFromString(purpose),
        orderId: orderId,
        network: network,
        address: address,
        redirectUrl: redirectUrl,
        amountUsd: amountUsd,
        receivedUsd: receivedUsd,
        status: paymentIntentStatusFromString(status),
        txHash: txHash,
        createdAt: createdAt == null ? null : DateTime.tryParse(createdAt!),
        expiresAt: expiresAt == null ? null : DateTime.tryParse(expiresAt!),
      );
}

/// Hand-written DTO for the payment feature-gate config.
class PaymentConfigDto {
  const PaymentConfigDto({
    this.usdtEnabled = false,
    this.whishEnabled = false,
    this.network = 'trc20',
    this.expiryMinutes = 30,
  });

  final bool usdtEnabled;
  final bool whishEnabled;
  final String network;
  final int expiryMinutes;

  factory PaymentConfigDto.fromJson(Map<String, dynamic> json) =>
      PaymentConfigDto(
        usdtEnabled: json['usdtEnabled'] as bool? ?? false,
        whishEnabled: json['whishEnabled'] as bool? ?? false,
        network: json['network'] as String? ?? 'trc20',
        expiryMinutes: (json['expiryMinutes'] as num?)?.toInt() ?? 30,
      );

  PaymentConfig toEntity() => PaymentConfig(
        usdtEnabled: usdtEnabled,
        whishEnabled: whishEnabled,
        network: network,
        expiryMinutes: expiryMinutes,
      );
}
