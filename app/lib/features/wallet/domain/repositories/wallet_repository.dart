import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/wallet.dart';

abstract interface class WalletRepository {
  /// The customer's wallet (balance + newest-first ledger).
  Future<Either<Failure, Wallet>> getWallet();

  /// Files a top-up request. Returns the pending request (credited only when
  /// an admin approves).
  Future<Either<Failure, TopUpRequest>> topUp(TopUpInput input);

  /// The customer's top-up request history, newest first.
  Future<Either<Failure, List<TopUpRequest>>> listTopUps();
}
