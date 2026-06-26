import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/kyc.dart';
import '../repositories/kyc_repository.dart';

class GetKycStatus {
  const GetKycStatus(this._repository);

  final KycRepository _repository;

  Future<Either<Failure, KycStatus>> call() => _repository.getStatus();
}
