import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../../../../core/storage/token_store.dart';
import '../../domain/entities/user.dart';
import '../../domain/repositories/auth_repository.dart';
import '../datasources/auth_remote_data_source.dart';

class AuthRepositoryImpl implements AuthRepository {
  const AuthRepositoryImpl(this._remote, this._tokenStore);

  final AuthRemoteDataSource _remote;
  final TokenStore _tokenStore;

  @override
  Future<Either<Failure, User>> login({
    required String email,
    required String password,
  }) async {
    try {
      final dto = await _remote.login(email: email, password: password);
      await _tokenStore.save(
        access: dto.accessToken,
        refresh: dto.refreshToken,
      );
      return Right(dto.user.toEntity());
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<void> logout() => _tokenStore.clear();
}
