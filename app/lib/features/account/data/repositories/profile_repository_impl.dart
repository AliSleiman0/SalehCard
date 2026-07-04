import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../../../auth/domain/entities/saved_player_id.dart';
import '../../../auth/domain/entities/user.dart';
import '../../domain/repositories/profile_repository.dart';
import '../datasources/profile_remote_data_source.dart';

class ProfileRepositoryImpl implements ProfileRepository {
  const ProfileRepositoryImpl(this._remote);

  final ProfileRemoteDataSource _remote;

  @override
  Future<Either<Failure, User>> getProfile() async {
    try {
      final dto = await _remote.getMe();
      return Right(dto.toEntity());
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<Either<Failure, User>> updateProfile({
    String? locale,
    List<SavedPlayerId>? savedPlayerIds,
  }) async {
    try {
      final dto = await _remote.updateMe(
        locale: locale,
        savedPlayerIds: savedPlayerIds,
      );
      return Right(dto.toEntity());
    } catch (error) {
      return Left(mapError(error));
    }
  }
}
