import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../repositories/kyc_repository.dart';

class UploadKycDocument {
  const UploadKycDocument(this._repository);

  final KycRepository _repository;

  Future<Either<Failure, String>> call(String filePath) =>
      _repository.uploadDocument(filePath);
}
