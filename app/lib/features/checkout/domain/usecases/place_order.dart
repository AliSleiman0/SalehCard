import 'package:fpdart/fpdart.dart' hide Order;

import '../../../../core/error/failure.dart';
import '../entities/order.dart';
import '../repositories/order_repository.dart';

class PlaceOrder {
  const PlaceOrder(this._repository);

  final OrderRepository _repository;

  Future<Either<Failure, Order>> call(
    PlaceOrderInput input, {
    required String idempotencyKey,
  }) {
    return _repository.placeOrder(input, idempotencyKey: idempotencyKey);
  }
}
