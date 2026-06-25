import 'package:fpdart/fpdart.dart' hide Order;

import '../../../../core/error/failure.dart';
import '../entities/order.dart';

abstract interface class OrderRepository {
  /// Places an order. [idempotencyKey] is sent on the `Idempotency-Key` header so
  /// the backend dedupes retries of the same checkout attempt.
  Future<Either<Failure, Order>> placeOrder(
    PlaceOrderInput input, {
    required String idempotencyKey,
  });

  Future<Either<Failure, Order>> getOrder(String id);

  /// Customer order history (newest first). Used by the Orders module.
  Future<Either<Failure, List<Order>>> listOrders();
}
