import 'package:dio/dio.dart';

import '../../../../core/network/api_envelope.dart';
import '../dtos/payment_intent_dto.dart';

class PaymentRemoteDataSource {
  const PaymentRemoteDataSource(this._dio);

  final Dio _dio;

  /// GET /payments/config → the USDT feature gate.
  Future<PaymentConfigDto> getConfig() async {
    final response = await _dio.get<dynamic>('/payments/config');
    return PaymentConfigDto.fromJson(unwrap(response) as Map<String, dynamic>);
  }

  /// POST /payments/usdt/topup-intents → a new pending on-chain top-up intent.
  /// [idempotencyKey] dedupes retries of the same attempt; an empty [network]
  /// lets the server pick its default.
  Future<PaymentIntentDto> createTopUpIntent(
    double amount, {
    required String idempotencyKey,
    String network = '',
  }) async {
    final response = await _dio.post<dynamic>(
      '/payments/usdt/topup-intents',
      data: {'amount': amount, if (network.isNotEmpty) 'network': network},
      options: Options(headers: {'Idempotency-Key': idempotencyKey}),
    );
    return PaymentIntentDto.fromJson(unwrap(response) as Map<String, dynamic>);
  }

  /// GET /payments/intents/{id} → the current intent state (polled).
  Future<PaymentIntentDto> getIntent(String id) async {
    final response = await _dio.get<dynamic>('/payments/intents/$id');
    return PaymentIntentDto.fromJson(unwrap(response) as Map<String, dynamic>);
  }
}
