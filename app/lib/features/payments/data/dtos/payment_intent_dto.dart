import '../../domain/entities/payment_intent.dart';

/// Hand-written DTO (no build_runner) for a payment intent — mirrors the
/// backend `IntentView`.
class PaymentIntentDto {
  const PaymentIntentDto({
    required this.id,
    this.purpose = '',
    this.orderId,
    this.network = 'trc20',
    this.address = '',
    this.amountUsd = 0,
    this.receivedUsd = 0,
    this.status = '',
    this.txHash = '',
    this.createdAt,
    this.expiresAt,
  });

  final String id;
  final String purpose;
  final String? orderId;
  final String network;
  final String address;
  final double amountUsd;
  final double receivedUsd;
  final String status;
  final String txHash;
  final String? createdAt;
  final String? expiresAt;

  factory PaymentIntentDto.fromJson(Map<String, dynamic> json) =>
      PaymentIntentDto(
        id: json['id'] as String? ?? '',
        purpose: json['purpose'] as String? ?? '',
        orderId: json['orderId'] as String?,
        network: json['network'] as String? ?? 'trc20',
        address: json['address'] as String? ?? '',
        amountUsd: (json['amountUsd'] as num?)?.toDouble() ?? 0,
        receivedUsd: (json['receivedUsd'] as num?)?.toDouble() ?? 0,
        status: json['status'] as String? ?? '',
        txHash: json['txHash'] as String? ?? '',
        createdAt: json['createdAt'] as String?,
        expiresAt: json['expiresAt'] as String?,
      );

  PaymentIntent toEntity() => PaymentIntent(
        id: id,
        purpose: paymentPurposeFromString(purpose),
        orderId: orderId,
        network: network,
        address: address,
        amountUsd: amountUsd,
        receivedUsd: receivedUsd,
        status: paymentIntentStatusFromString(status),
        txHash: txHash,
        createdAt: createdAt == null ? null : DateTime.tryParse(createdAt!),
        expiresAt: expiresAt == null ? null : DateTime.tryParse(expiresAt!),
      );
}

/// Hand-written DTO for the USDT feature-gate config.
class PaymentConfigDto {
  const PaymentConfigDto({
    this.usdtEnabled = false,
    this.network = 'trc20',
    this.networks = const [],
    this.expiryMinutes = 30,
  });

  final bool usdtEnabled;
  final String network;
  final List<String> networks;
  final int expiryMinutes;

  factory PaymentConfigDto.fromJson(Map<String, dynamic> json) {
    final network = json['network'] as String? ?? 'trc20';
    return PaymentConfigDto(
      usdtEnabled: json['usdtEnabled'] as bool? ?? false,
      network: network,
      // Servers that predate multi-network omit the list — fall back to the
      // single default network.
      networks: (json['networks'] as List<dynamic>?)
              ?.whereType<String>()
              .toList() ??
          [network],
      expiryMinutes: (json['expiryMinutes'] as num?)?.toInt() ?? 30,
    );
  }

  PaymentConfig toEntity() => PaymentConfig(
        usdtEnabled: usdtEnabled,
        network: network,
        networks: networks.isEmpty ? [network] : networks,
        expiryMinutes: expiryMinutes,
      );
}
