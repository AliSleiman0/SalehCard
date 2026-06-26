import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/user.dart';
import '../repositories/auth_repository.dart';

/// Verifies a one-time passcode and signs the user in (creating the account on
/// first sign-in). A non-null [password] is set on a brand-new account (signup).
class VerifyOtp {
  const VerifyOtp(this._repository);

  final AuthRepository _repository;

  Future<Either<Failure, User>> call({
    required String phone,
    required String code,
    String? password,
  }) {
    return _repository.verifyOtp(phone: phone, code: code, password: password);
  }
}
