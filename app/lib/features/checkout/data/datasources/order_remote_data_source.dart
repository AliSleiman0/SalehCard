import 'package:dio/dio.dart';

import '../../../../core/network/api_envelope.dart';
import '../../../payments/data/dtos/payment_intent_dto.dart';
import '../../domain/entities/order.dart';
import '../dtos/order_dto.dart';

class OrderRemoteDataSource {
  const OrderRemoteDataSource(this._dio);

  final Dio _dio;

  /// POST /orders. The price is derived server-side; [idempotencyKey] dedupes
  /// retries of the same checkout attempt via the `Idempotency-Key` header.
  /// A USDT order comes back still-pending with an embedded `paymentIntent`
  /// (deposit address + amount) attached to the returned entity.
  Future<Order> placeOrder(
    PlaceOrderInput input, {
    required String idempotencyKey,
  }) async {
    final response = await _dio.post<dynamic>(
      '/orders',
      data: _toRequestJson(input),
      options: Options(headers: {'Idempotency-Key': idempotencyKey}),
    );
    final map = unwrap(response) as Map<String, dynamic>;
    final order = OrderDto.fromJson(map).toEntity();
    final intent = map['paymentIntent'];
    if (intent is Map<String, dynamic>) {
      return order.withPaymentIntent(PaymentIntentDto.fromJson(intent).toEntity());
    }
    return order;
  }

  Future<OrderDto> getOrder(String id) async {
    final response = await _dio.get<dynamic>('/orders/$id');
    return OrderDto.fromJson(unwrap(response) as Map<String, dynamic>);
  }

  Future<List<OrderDto>> listOrders() async {
    final response = await _dio.get<dynamic>('/orders');
    final data = unwrap(response) as List<dynamic>;
    return data
        .map((e) => OrderDto.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Map<String, dynamic> _toRequestJson(PlaceOrderInput input) {
    return {
      'items': [
        for (final line in input.items)
          {
            'productId': line.productId,
            'variantId': line.variantId,
            'qty': line.qty,
            if (line.playerId != null && line.playerId!.isNotEmpty)
              'playerId': line.playerId,
            if (line.recipient != null)
              'recipient': {
                'name': line.recipient!.name,
                'country': line.recipient!.country,
                'detail': line.recipient!.detail,
              },
            if (line.fields.isNotEmpty)
              'fields': [
                for (final f in line.fields) {'key': f.key, 'value': f.value},
              ],
          },
      ],
      'currency': input.currency,
      'paymentMethod': input.paymentMethod,
      if (input.promoCode != null && input.promoCode!.isNotEmpty)
        'promoCode': input.promoCode,
    };
  }
}
