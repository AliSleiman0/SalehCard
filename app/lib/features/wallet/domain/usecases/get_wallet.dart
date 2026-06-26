import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/wallet.dart';
import '../repositories/wallet_repository.dart';

class GetWallet {
  const GetWallet(this._repository);

  final WalletRepository _repository;

  Future<Either<Failure, Wallet>> call() => _repository.getWallet();
}
