import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/wallet.dart';
import '../repositories/wallet_repository.dart';

class TopUp {
  const TopUp(this._repository);

  final WalletRepository _repository;

  Future<Either<Failure, WalletTx>> call(TopUpInput input) =>
      _repository.topUp(input);
}
