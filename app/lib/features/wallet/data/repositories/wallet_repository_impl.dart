import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../../domain/entities/wallet.dart';
import '../../domain/repositories/wallet_repository.dart';
import '../datasources/wallet_remote_data_source.dart';

class WalletRepositoryImpl implements WalletRepository {
  const WalletRepositoryImpl(this._remote);

  final WalletRemoteDataSource _remote;

  @override
  Future<Either<Failure, Wallet>> getWallet() async {
    try {
      final dto = await _remote.getWallet();
      return Right(dto.toEntity());
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<Either<Failure, TopUpRequest>> topUp(TopUpInput input) async {
    try {
      final dto = await _remote.topUp(input);
      return Right(dto.toEntity());
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<Either<Failure, List<TopUpRequest>>> listTopUps() async {
    try {
      final dtos = await _remote.listTopUps();
      return Right([for (final dto in dtos) dto.toEntity()]);
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<Either<Failure, List<TopUpMethod>>> listMethods() async {
    try {
      final dtos = await _remote.listMethods();
      return Right([for (final dto in dtos) dto.toEntity()]);
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<Either<Failure, String>> uploadDocument(String filePath) async {
    try {
      return Right(await _remote.uploadDocument(filePath));
    } catch (error) {
      return Left(mapError(error));
    }
  }
}
