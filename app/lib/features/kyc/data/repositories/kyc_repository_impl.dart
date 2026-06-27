import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../../domain/entities/kyc.dart';
import '../../domain/repositories/kyc_repository.dart';
import '../datasources/kyc_remote_data_source.dart';

class KycRepositoryImpl implements KycRepository {
  const KycRepositoryImpl(this._remote);

  final KycRemoteDataSource _remote;

  @override
  Future<Either<Failure, KycProfile>> getProfile() async {
    try {
      return Right(await _remote.getProfile());
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<Either<Failure, KycProfile>> submit(KycSubmission submission) async {
    try {
      return Right(await _remote.submit(submission));
    } catch (error) {
      return Left(mapError(error));
    }
  }
}
