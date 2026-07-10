import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../../domain/entities/payment_intent.dart';
import '../../domain/repositories/payment_repository.dart';
import '../datasources/payment_remote_data_source.dart';

class PaymentRepositoryImpl implements PaymentRepository {
  const PaymentRepositoryImpl(this._remote);

  final PaymentRemoteDataSource _remote;

  @override
  Future<Either<Failure, PaymentConfig>> getConfig() async {
    try {
      return Right((await _remote.getConfig()).toEntity());
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<Either<Failure, PaymentIntent>> createTopUpIntent(
    double amount, {
    required String idempotencyKey,
    String network = '',
  }) async {
    try {
      final dto = await _remote.createTopUpIntent(
        amount,
        idempotencyKey: idempotencyKey,
        network: network,
      );
      return Right(dto.toEntity());
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<Either<Failure, PaymentIntent>> getIntent(String id) async {
    try {
      return Right((await _remote.getIntent(id)).toEntity());
    } catch (error) {
      return Left(mapError(error));
    }
  }
}
