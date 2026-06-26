import 'package:fpdart/fpdart.dart' hide Order;

import '../../../../core/error/failure.dart';
import '../entities/order.dart';
import '../repositories/order_repository.dart';

class GetOrder {
  const GetOrder(this._repository);

  final OrderRepository _repository;

  Future<Either<Failure, Order>> call(String id) => _repository.getOrder(id);
}
