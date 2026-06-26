import 'package:fpdart/fpdart.dart' hide Order;

import '../../../../core/error/failure.dart';
import '../entities/order.dart';
import '../repositories/order_repository.dart';

class GetOrders {
  const GetOrders(this._repository);

  final OrderRepository _repository;

  Future<Either<Failure, List<Order>>> call() => _repository.listOrders();
}
