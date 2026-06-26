import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../../domain/entities/kyc.dart';
import '../../domain/repositories/kyc_repository.dart';

/// The status a freshly-created stub starts in. Flip this single line to
/// `KycStatus.verified` / `KycStatus.rejected` / `KycStatus.pending` to preview
/// the other status cards during device verification.
const KycStatus _kInitialStatus = KycStatus.unverified;

/// Display-only stub for KYC verification.
///
/// TODO(backend): replace stub with real KYC endpoints when they exist. Holds
/// the status in memory: [getStatus] returns the current value and [submit]
/// flips it to [KycStatus.pending] (simulating "submitted, under review") so the
/// form → pending flow works end-to-end without a backend. This is stateful, so
/// the provider must keep a single instance alive (see `kycRepositoryProvider`).
class KycRepositoryStub implements KycRepository {
  KycRepositoryStub();

  KycStatus _status = _kInitialStatus;

  @override
  Future<Either<Failure, KycStatus>> getStatus() async {
    return Right(_status);
  }

  @override
  Future<Either<Failure, KycStatus>> submit(KycSubmission submission) async {
    // TODO(backend): POST the submission to a real KYC endpoint. For now we just
    // move to "pending" so the UI can render the under-review card.
    _status = KycStatus.pending;
    return Right(_status);
  }
}
