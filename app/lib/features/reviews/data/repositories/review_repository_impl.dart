import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../../domain/entities/review.dart';
import '../../domain/repositories/review_repository.dart';
import '../datasources/review_remote_data_source.dart';

class ReviewRepositoryImpl implements ReviewRepository {
  const ReviewRepositoryImpl(this._remote);

  final ReviewRemoteDataSource _remote;

  @override
  Future<Either<Failure, Unit>> submit(ReviewDraft draft) async {
    try {
      await _remote.submit(draft);
      return const Right(unit);
    } catch (error) {
      return Left(mapError(error));
    }
  }

  @override
  Future<Either<Failure, MyReview>> myReview(String productId) async {
    try {
      return Right(await _remote.myReview(productId));
    } catch (error) {
      return Left(mapError(error));
    }
  }
}
