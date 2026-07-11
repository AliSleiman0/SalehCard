import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../repositories/profile_repository.dart';

class DeleteAccount {
  const DeleteAccount(this._repository);

  final ProfileRepository _repository;

  Future<Either<Failure, Unit>> call() => _repository.deleteAccount();
}
