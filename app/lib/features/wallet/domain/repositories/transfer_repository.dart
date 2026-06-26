import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';

/// Peer-to-peer money transfer. NOTE: there is no backend endpoint for this yet,
/// so the only implementation today is a stub (see [TransferRepositoryStub]).
/// Kept as a real interface so swapping in an HTTP impl later is one file.
abstract interface class TransferRepository {
  Future<Either<Failure, Unit>> sendMoney({
    required String recipient,
    required double amount,
    String? note,
  });
}
