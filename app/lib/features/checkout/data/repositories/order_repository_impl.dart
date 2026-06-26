import 'package:fpdart/fpdart.dart' hide Order;

import '../../../../core/error/failure.dart';
import '../../domain/entities/order.dart';
import '../../domain/repositories/order_repository.dart';
import '../datasources/order_remote_data_source.dart';

class OrderRepositoryImpl implements OrderRepository {
  const OrderRepositoryImpl(this._remote);

  final OrderRemoteDataSource _remote;

  @override
  Future<Either<Failure, Order>> placeOrder(
    PlaceOrderInput input, {
    required String idempotencyKey,
  }) async {
    try {
      final dto =
          await _remote.placeOrder(input, idempotencyKey: idempotencyKey);
      return Right(dto.toEntity());
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<Either<Failure, Order>> getOrder(String id) async {
    try {
      final dto = await _remote.getOrder(id);
      return Right(dto.toEntity());
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<Either<Failure, List<Order>>> listOrders() async {
    try {
      final dtos = await _remote.listOrders();
      return Right(dtos.map((d) => d.toEntity()).toList());
    } catch (error) {
      return Left(mapError(error));
    }
  }
}
