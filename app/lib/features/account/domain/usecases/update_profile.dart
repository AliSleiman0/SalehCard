import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../../../auth/domain/entities/user.dart';
import '../repositories/profile_repository.dart';

class UpdateProfile {
  const UpdateProfile(this._repository);

  final ProfileRepository _repository;

  Future<Either<Failure, User>> call({
    String? locale,
    List<String>? savedPlayerIds,
  }) =>
      _repository.updateProfile(locale: locale, savedPlayerIds: savedPlayerIds);
}
