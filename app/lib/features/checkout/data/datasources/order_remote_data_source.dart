import 'package:dio/dio.dart';

import '../../../../core/network/api_envelope.dart';
import '../../domain/entities/order.dart';
import '../dtos/order_dto.dart';

class OrderRemoteDataSource {
  const OrderRemoteDataSource(this._dio);

  final Dio _dio;

  /// POST /orders. The price is derived server-side; [idempotencyKey] dedupes
  /// retries of the same checkout attempt via the `Idempotency-Key` header.
  Future<OrderDto> placeOrder(
    PlaceOrderInput input, {
    required String idempotencyKey,
  }) async {
    final response = await _dio.post<dynamic>(
      '/orders',
      data: _toRequestJson(input),
      options: Options(headers: {'Idempotency-Key': idempotencyKey}),
    );
    return OrderDto.fromJson(unwrap(response) as Map<String, dynamic>);
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
          },
      ],
      'currency': input.currency,
      'paymentMethod': input.paymentMethod,
      if (input.promoCode != null && input.promoCode!.isNotEmpty)
        'promoCode': input.promoCode,
    };
  }
}
