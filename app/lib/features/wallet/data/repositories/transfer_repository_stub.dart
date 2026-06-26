import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../../domain/repositories/transfer_repository.dart';

/// Display-only stub for peer-to-peer transfers.
///
/// TODO(backend): no peer-transfer endpoint exists yet. This always returns a
/// [ServerFailure] carrying a "coming soon" message so the UI can render the
/// full send-money form without faking a real transfer. Replace with an HTTP
/// implementation once the backend exposes a transfer endpoint.
class TransferRepositoryStub implements TransferRepository {
  const TransferRepositoryStub();

  @override
  Future<Either<Failure, Unit>> sendMoney({
    required String recipient,
    required double amount,
    String? note,
  }) async {
    // TODO(backend): wire to POST /wallet/send (or equivalent) when it exists.
    return const Left(
      ServerFailure('NOT_IMPLEMENTED', 'Send money is coming soon'),
    );
  }
}
