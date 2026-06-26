import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/wallet.dart';

abstract interface class WalletRepository {
  /// The customer's wallet (balance + newest-first ledger).
  Future<Either<Failure, Wallet>> getWallet();

  /// Tops up the wallet. Returns the newly-credited transaction.
  Future<Either<Failure, WalletTx>> topUp(TopUpInput input);
}
