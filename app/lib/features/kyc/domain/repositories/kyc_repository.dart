import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/kyc.dart';

/// KYC verification. NOTE: there is no backend endpoint for this yet, so the
/// only implementation today is a stub (see [KycRepositoryStub]). Kept as a real
/// interface so swapping in an HTTP impl later is one file.
abstract interface class KycRepository {
  /// The customer's current verification status.
  Future<Either<Failure, KycStatus>> getStatus();

  /// Submits the verification form. Returns the new status — today always
  /// [KycStatus.pending] (review is simulated).
  Future<Either<Failure, KycStatus>> submit(KycSubmission submission);
}
