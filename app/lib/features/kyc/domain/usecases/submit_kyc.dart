import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/kyc.dart';
import '../repositories/kyc_repository.dart';

class SubmitKyc {
  const SubmitKyc(this._repository);

  final KycRepository _repository;

  Future<Either<Failure, KycProfile>> call(KycSubmission submission) =>
      _repository.submit(submission);
}
