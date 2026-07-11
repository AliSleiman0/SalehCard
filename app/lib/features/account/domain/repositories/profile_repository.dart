import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../../../auth/domain/entities/saved_player_id.dart';
import '../../../auth/domain/entities/user.dart';

abstract interface class ProfileRepository {
  /// The authenticated user's profile (`GET /users/me`).
  Future<Either<Failure, User>> getProfile();

  /// Updates the editable profile fields (`PATCH /users/me`). Both are optional
  /// — pass only what changed. Returns the updated user.
  Future<Either<Failure, User>> updateProfile({
    String? locale,
    List<SavedPlayerId>? savedPlayerIds,
  });

  /// Permanently deletes the authenticated account (`DELETE /users/me`).
  Future<Either<Failure, Unit>> deleteAccount();
}
