import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/payment_intent.dart';

/// Read/create surface for on-chain USDT payments.
abstract class PaymentRepository {
  Future<Either<Failure, PaymentConfig>> getConfig();

  Future<Either<Failure, PaymentIntent>> createTopUpIntent(
    double amount, {
    required String idempotencyKey,
  });

  Future<Either<Failure, PaymentIntent>> getIntent(String id);
}
