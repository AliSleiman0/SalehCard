import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/review.dart';
import '../repositories/review_repository.dart';

class GetMyReview {
  const GetMyReview(this._repository);

  final ReviewRepository _repository;

  Future<Either<Failure, MyReview>> call(String productId) =>
      _repository.myReview(productId);
}
