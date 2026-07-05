import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/kyc.dart';

/// KYC verification, backed by the API (`/api/v1/kyc`).
abstract interface class KycRepository {
  /// The customer's current verification profile (status + optional rejection
  /// reason). A user with no submission is [KycStatus.unverified].
  Future<Either<Failure, KycProfile>> getProfile();

  /// Submits the verification form. Returns the new profile (pending review).
  Future<Either<Failure, KycProfile>> submit(KycSubmission submission);

  /// Uploads one document photo (multipart); returns its public URL, which the
  /// form places in [KycSubmission.documentFrontUrl]/[KycSubmission.documentBackUrl].
  Future<Either<Failure, String>> uploadDocument(String filePath);
}
