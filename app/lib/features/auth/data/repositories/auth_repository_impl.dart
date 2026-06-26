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
  Future<Either<Failure, User>> loginByPhone({
    required String phone,
    required String password,
  }) async {
    try {
      final dto = await _remote.loginByPhone(phone: phone, password: password);
      await _tokenStore.save(access: dto.accessToken, refresh: dto.refreshToken);
      return Right(dto.user.toEntity());
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<Either<Failure, Unit>> requestOtp({required String phone}) async {
    try {
      await _remote.requestOtp(phone: phone);
      return const Right(unit);
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<Either<Failure, User>> verifyOtp({
    required String phone,
    required String code,
    String? password,
  }) async {
    try {
      final dto = await _remote.verifyOtp(
        phone: phone,
        code: code,
        password: password,
      );
      await _tokenStore.save(access: dto.accessToken, refresh: dto.refreshToken);
      return Right(dto.user.toEntity());
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<void> logout() => _tokenStore.clear();
}
