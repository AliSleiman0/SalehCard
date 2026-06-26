import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../repositories/auth_repository.dart';

/// Sends a one-time passcode to a phone number.
class RequestOtp {
  const RequestOtp(this._repository);

  final AuthRepository _repository;

  Future<Either<Failure, Unit>> call({required String phone}) {
    return _repository.requestOtp(phone: phone);
  }
}
