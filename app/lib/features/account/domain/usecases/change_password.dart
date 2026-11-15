import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../repositories/profile_repository.dart';

class ChangePassword {
  const ChangePassword(this._repository);

  final ProfileRepository _repository;

  Future<Either<Failure, Unit>> call({
    required String currentPassword,
    required String newPassword,
  }) => _repository.changePassword(
    currentPassword: currentPassword,
    newPassword: newPassword,
  );
}
