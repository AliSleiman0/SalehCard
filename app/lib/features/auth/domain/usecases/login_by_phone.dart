import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/user.dart';
import '../repositories/auth_repository.dart';

/// Logs a user in with phone + password.
class LoginByPhone {
  const LoginByPhone(this._repository);

  final AuthRepository _repository;

  Future<Either<Failure, User>> call({
    required String phone,
    required String password,
  }) {
    return _repository.loginByPhone(phone: phone, password: password);
  }
}
